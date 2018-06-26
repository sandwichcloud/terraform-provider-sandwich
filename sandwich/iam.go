package sandwich

import (
	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/sandwichcloud/deli-cli/api"
	"github.com/sandwichcloud/deli-cli/api/client"
)

var iamMutexKV = mutexkv.NewMutexKV()

type iamPolicyModifyFunc func(policy *api.Policy)

func iamReadModifyWrite(mutexKey string, policyClient client.PolicyClientInterface, modify iamPolicyModifyFunc) error {
	iamMutexKV.Lock(mutexKey)
	defer iamMutexKV.Unlock(mutexKey)

	policy, err := policyClient.Get()
	if err != nil {
		return err
	}
	modify(policy)
	err = policyClient.Set(*policy)
	if err != nil {
		return err
	}
	return nil
}
