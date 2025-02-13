/*
Copyright AppsCode Inc. and Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import meta_util "kmodules.xyz/client-go/meta"

func (r *Recommendation) IsAwaitingOrProgressingRecommendation() bool {
	return r.IsAwaitingRecommendation() || r.IsProgressingRecommendation()
}

func (r *Recommendation) IsAwaitingRecommendation() bool {
	switch r.Status.Phase {
	case "":
		fallthrough
	case Pending:
		fallthrough
	case Waiting:
		return true
	default:
		return false
	}
}

func (r *Recommendation) IsProgressingRecommendation() bool {
	return r.Status.Phase == InProgress
}

func (r *Recommendation) MergeWithOffshootLabels(extraLabels ...map[string]string) map[string]string {
	return meta_util.OverwriteKeys(r.GetLabels(), extraLabels...)
}
