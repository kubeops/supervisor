Meeting Recording: https://youtu.be/rjvzK9jxcqI


Use-cases:
1. TLS cert renew
2. Volume Expansion
3. KubeDB catalog container image mismatch - Same version update
4. Patch version update / log4j security - update version
5. License renew

Admin:
Creates Maintenance Window

Cluster
Namespace
GK (crd type specific) for auto approaval
Parallelism

type AutoApproval struct {

}

Recommedation Generation:
- Application Specific (done by that app's ops-manager)
- generate name?
- deployment name?

Approval:
- App owner / cluster owner


Immediate
NextAvailable

Specific Window on Specific Date

Date 

Deadline


Cluster
Namespace

GK
G


apiVersion: 
kind: MaintenanceWindow
spec:
  isDefault: true
  
    days:
  	  sunday:
  	  - start: 6:00PM
  	    end: 10:00PM
  	  - start: 11:00PM
  	    end: 11:30PM
  	  friday:
  	  - start: 6:00PM
  	    end: 10:00PM
  	  - start: 11:00PM
  	    end: 11:30PM

	dates:
	  - start: 2022/01/18 6:00 PM
	    end: 2022/01/20 6:00 AM
	  - start: 2022/01/22 6:00 PM
	    end: 2022/01/24 6:00 AM

apiVersion: 
kind: ApprovalPolicy
spec:
  maintenanceWindow:
    kind:
    name:
  targets:
  - group: kubedb.com
    kind: MongoDB
    operation:
    - group: ops.kubedb.com
      kind: ReconfigureTLS
  - group: kubedb.com
    kind: Elasticsearch
    operation:
    - group: ops.kubedb.com
      kind: ReconfigureTLS
