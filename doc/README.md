## Supervisor

`Supervisor` is one of the key components of `KubeDB AutoOps` infrastructure. `KubeDB AutoOps` is a concept that is focusing to automate the day 2 life-cycle management for databases on Kubernetes. KubeDB manages the day 2 life-cycle for a Database using `OpsRequest`. But these processes are manual and users have to create those OpsRequest with immediate effect. By using KubeDB AutoOps, these processes can be made fully automated & configurable.

To achieve this, we introduced two KubeDB AutoOps components. One is `Recommendation Generator` which will generate Recommendation by inspecting KubeDB Database resources. And another one is `Supervisor` which will execute those recommendations in the specified `Maintenance Window`.

Supervisor watches the Recommendation created by KubeDB Ops-Manager and reconciled necessary actions on as per the Recommendation. It is responsible for applying the recommended operations immediately or at a specified schedule.

Supervisor have four major components:
1. Recommendation
2. Approval Policy
3. Maintenance Window
4. Cluster Maintenance Window

Here, are some of the key features that `KubeDB AutoOps` provides via Supervisor:

- **Targeted Recommendations:** Specifies the exact Kubernetes resource (e.g., a database) for which the recommendation applies.
- **Automated Operations:** Executes Kubernetes manifest definitions to perform the recommended actions.
- **Vulnerability Reporting:** Integrates vulnerability reports, including CVE fix information.
- **Approval Workflow:** Supports manual approval for sensitive operations.
- **Execution Rules:** Defines rules for determining the success, progress, and failure of operations.
- **Retry Mechanism:** Configures retry limits for failed operations.
- **Deadline Management:** Sets deadlines for operation execution.
- **Maintenance Window Integration:** Integrates with maintenance windows for controlled execution.
- **Parallelism Control:** Manages concurrent execution of recommendations.
- **Status Tracking:** Provides detailed status information, including approval status, execution phase, and conditions.
- **Outdated Detection:** Detects if a recommendation is outdated due to changes in the target resource.

Here's a sample `Recommendation` that can be generated for a MongoDB cluster. 

```yaml
apiVersion: supervisor.appscode.com/v1alpha1
kind: Recommendation
metadata:
  creationTimestamp: "2025-02-27T13:02:15Z"
  generation: 1
  labels:
    app.kubernetes.io/instance: mongo
    app.kubernetes.io/managed-by: kubedb.com
    app.kubernetes.io/type: rotate-tls
  name: mongo-x-mongodb-x-rotate-tls-8pjwow
  namespace: mg
  resourceVersion: "317434"
  uid: 4b35e948-2e15-4f04-be22-3c66a0c988fb
spec:
  backoffLimit: 5
  deadline: "2025-02-27T13:19:34Z"
  description: Recommending TLS certificate rotation,mongo-client-cert Certificate
    is going to be expire on 2025-02-27 13:24:34 +0000 UTC
  operation:
    apiVersion: ops.kubedb.com/v1alpha1
    kind: MongoDBOpsRequest
    metadata:
      name: rotate-tls
      namespace: mg
    spec:
      databaseRef:
        name: mongo
      tls:
        rotateCertificates: true
      type: ReconfigureTLS
    status: {}
  recommender:
    name: kubedb-ops-manager
  rules:
    failed: has(self.status) && has(self.status.phase) && self.status.phase == 'Failed'
    inProgress: has(self.status) && has(self.status.phase) && self.status.phase ==
      'Progressing'
    success: has(self.status) && has(self.status.phase) && self.status.phase == 'Successful'
  target:
    apiGroup: kubedb.com
    kind: MongoDB
    name: mongo
status:
  approvalStatus: Pending
  failedAttempt: 0
  outdated: false
  parallelism: Namespace
  phase: Pending
  reason: WaitingForApproval
```

The `RecommendationSpec` defines the recommended operations and corresponding information of a `Recommendation` resource. Here are the available fields in a `RecommendationSpec`.

- **description:**
    * A human-readable string that explains the purpose and reason for this recommendation.
    * Useful for providing context to reviewers and operators.

- **vulnerabilityReport:**
    * Details about any vulnerabilities related to the recommendation.
    * Contains information about CVEs that will be fixed or remain after applying the recommendation.
    * Includes the status of the vulnerability report generation, messages, and detailed lists of vulnerabilities.
    * Helps in understanding the security implications of the recommendation.
- **target:**
    * A reference to the Kubernetes resource that the recommendation applies to.
    * Specifies the `APIGroup`, `Kind`, and `Name` of the target resource.
    * Ensures that the recommendation is applied to the correct resource.
- **operation:**
    * A Kubernetes object YAML that defines the operation to be performed.
    * This YAML should be a valid Kubernetes resource definition with `apiVersion`, `kind`, and `metadata` fields.
    * This is the core of the recommendation, specifying the actual change to be made.
- **recommender:**
    * A reference to the component that generated the recommendation.
- **deadline:**
    * A timestamp that specifies the deadline for executing the recommendation.
    * The recommendation should be executed before this deadline.
    * Used to enforce time-sensitive operations.
- **requireExplicitApproval:**
    * A boolean flag that indicates whether manual approval is required before executing the recommendation.
    * If `true`, the recommendation will not be executed without manual approval.
    * Provides a safety mechanism for critical operations.
- **rules:**
    * Defines rules for determining the success, progress, and failure of the operation.
    * Uses CEL (Common Expression Language) expressions to evaluate the status of the operation.
    * Consists of three fields: `Success`, `InProgress`, and `Failed`.
    * These rules are vital for the controller to properly manage the lifecycle of the recommendation.
- **backoffLimit:**
    * Specifies the number of retries before marking the recommendation as failed.
    * Defaults to 5.
    * If set to 0, the operation will be tried only once.
    * Increases the reliability of the operation execution.

The `RecommendationStatus` defines the observed state of a `Recommendation` resource. It provides information about the current state of the recommendation. Here are the available fields in `RecommendationStatus`.

- **approvalStatus:**
    * Specifies the approval status of the recommendation.
    * Possible values: `Pending`, `Approved`, `Rejected`.
    * Defaults to `Pending`.
    * Indicates whether the recommendation has been approved or rejected.
- **phase:**
    * Specifies the current phase of the recommendation execution.
    * Possible values: `Pending`, `Skipped`, `Waiting`, `InProgress`, `Succeeded`, `Failed`.
    * Provides a high-level overview of the recommendation's progress.
- **reason:**
    * A message that provides details about the current phase of the recommendation.
    * Helps in diagnosing issues and understanding the recommendation's state.
- **reviewer:**
    * Details about the reviewer who approved or rejected the recommendation.
    * Specifies the `Kind`, `APIGroup`, `Name`, and `Namespace` of the reviewer.
    * Provides an audit trail of the review process.
- **comments:**
    * Comments provided by the reviewer during the approval or rejection process.
    * Useful for providing additional context and rationale.
- **reviewTimestamp:**
    * The timestamp of the review.
    * Indicates when the recommendation was approved or rejected.
- **approvedWindow:**
    * Specifies the time window configuration for executing the recommendation.
    * Integrates with maintenance windows.
    * Allows for controlled execution during specific time periods.
- **parallelism:**
    * Controls the concurrent execution of recommendations.
    * Possible values: `Namespace`, `Target`, `TargetAndNamespace`.
    * Defaults to `Namespace`.
    * Helps in managing resource contention.
- **observedGeneration:**
    * The most recent generation observed for this resource.
    * Used to detect changes in the recommendation specification.
- **conditions:**
    * Conditions applied to the recommendation.
    * Provides detailed information about the recommendation's state.
- **outdated:**
    * Indicates whether the recommendation is outdated.
    * If true, the recommendation will not be executed.
    * Defaults to false.
    * Prevents execution of obsolete recommendations.
- **createdOperationRef:**
    * Holds the created operation name.
    * Used to track the ops-request resource created by the recommendation.
- **failedAttempt:**
    * Holds the number of times the operation has failed.
    * Defaults to 0.
    * Used in conjunction with `BackoffLimit` to manage retries.

By understanding these fields, operators can effectively manage and automate database maintenance and administrative operations using the `Recommendation` CRD.

Let's discuss `ApprovalPolicy` resource. It is used to:
- **Automate Recommendation Approvals:** Automatically approve `Recommendation` resources based on predefined criteria.
- **Streamline Maintenance Operations:** Reduce manual intervention for routine tasks, ensuring timely execution.
- **Enhance Operational Efficiency:** Improve the speed and efficiency of database maintenance workflows.
- **Provide Granular Control:** Define policies based on target resources and specific operations.
Now, Here's how recommendation `ApprovalPolicy` looks like. 

```yaml
apiVersion: supervisor.appscode.com/v1alpha1
kind: ApprovalPolicy
metadata:
  name: mongo-policy
  namespace: demo
maintenanceWindowRef:
  name: default-mw
targets:
  - group: kubedb.com
    kind: MongoDB
    operations:
      - group: ops.kubedb.com
        kind: MongoDBOpsRequest
```

Here's the fields available in a `ApprovalPolicy` Object:

* **maintenanceWindowRef:**
    * Specifies the reference to the `MaintenanceWindow` resource.
    * Recommendations will be automatically approved if they fall within this maintenance window.

* **targets:**
    * Specifies a list of target resources for which the approval policy will be effective.
    * If this field is omitted, the policy applies to all target resources within the specified maintenance window.

- **targetRef:**
  - **group:**
      * The API group of the target resource.
  - **kind:**
      * The kind of the target resource.
  - **operations:**
      * A list of specific operations (defined by `Group` and `Kind`) that should be automatically approved for the target resource.

The `ApprovalPolicy` CRD can be used to:

* Automatically approve routine database upgrades during scheduled maintenance windows.
* Automatically apply security patches to specific database instances.
* Streamline the execution of performance optimization tasks.
* Define policies that apply to all databases within a namespace.
* Define policies that apply to specific operations on specific databases.

The `MaintenanceWindow` object is designed to manage and schedule maintenance periods for Kubernetes resources, particularly databases. It allows administrators to define specific time windows during which maintenance operations, such as upgrades or rotation, can be performed. This ensures that critical operations are carried out during planned downtimes, minimizing disruptions to applications.

Here's some key features of `MaintenanceWindow`:

* **Default Maintenance Window:** Option to mark a window as the default.
* **Time Zone Support:** Configuration for time zones, including UTC and local server time.
* **Day-Based Scheduling:** Define maintenance windows for specific days of the week.
* **Date-Based Scheduling:** Define maintenance windows for specific dates.
* **Time Window Definition:** Specify start and end times for maintenance periods.

Let's checkout a sample `MaintenanceWindow`:

```yaml
apiVersion: supervisor.appscode.com/v1alpha1
kind: MaintenanceWindow
metadata:
  name: default-mw
  namespace: demo
spec:
  isDefault: true
  timeZone: Asia/Dhaka
  days:
    Monday:
      - start: 10:40AM
        end: 7:00PM
  dates:
    - start: 2022-01-24T00:00:18Z
      end: 2022-01-24T23:41:18Z

```

Here's how the fields can be used in a `MaintenanceWindow`:

- **`isDefault` (bool, optional):**
    * Indicates whether this maintenance window is the default.
    * If set to `true`, this window will be used when no other specific maintenance window is specified.
    * Example: `isDefault: true`

- **timezone:**
    * Specifies the time zone for the maintenance window.
    * If not set, empty (""), or "UTC", the given times and dates are considered UTC.
    * If set to "Local", the times and dates are considered as the server's local time zone.
    * Otherwise, it should be a location name from the IANA Time Zone database (e.g., "Asia/Dhaka", "America/New_York").
    * Example: `timezone: "Asia/Dhaka"`

- **days:**
    * Defines maintenance windows for specific days of the week.
    * Consists of a map where the keys are `DayOfWeek` (e.g., "Monday", "Tuesday") and the values are lists of `TimeWindow`.
    * Example:
        ```yaml
        days:
          Monday:
            - start: "10:00AM"
              end: "07:00PM"
          Wednesday:
            - start: "02:00PM"
              end: "06:00PM"
        ```

- **Dates:**
    * Defines maintenance windows for specific dates.
    * Dates should be in UTC format: `YYYY-MM-DDThh.mm.ssZ`.
    * Example:
        ```yaml
        dates:
          - start: "2024-12-25T00:00:00Z"
            end: "2024-12-25T23:59:59Z"
          - start: "2025-01-01T00:00:00Z"
            end: "2025-01-01T23:59:59Z"
        ```

The `MaintenanceWindow` CRD can be used to:

* Schedule weekly maintenance windows for database upgrades.
* Define specific date ranges for major maintenance activities.
* Set default maintenance windows for automatic application.
* Manage maintenance schedules for resources across different time zones.
* Integrate with `Recommendation` and `ApprovalPolicy` CRDs to automate maintenance tasks.

## Benefits

* Reduced manual effort for scheduling maintenance.
* Minimized application downtime through planned maintenance periods.
* Improved operational efficiency through automated scheduling.
* Enhanced control over maintenance activities.
* Consistent and repeatable maintenance schedules.