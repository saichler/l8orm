/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package persist

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8web"
)

// do executes a database write operation (POST, PUT, PATCH) with callback support.
// It follows the pattern: Before callbacks -> ORM write -> After callbacks.
// Returns an empty response on success, or an error response on failure.
func (this *OrmService) do(action ifs.Action, pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	pbBefore, cont := this.Before(action, pb, vnic)
	if !cont {
		return object.New(nil, &l8web.L8Empty{})
	}
	if pbBefore != nil {
		if pbBefore.Error() != nil {
			return pbBefore
		}
		pb = pbBefore
	}

	err := this.orm.Write(action, pb, vnic.Resources())

	if err != nil {
		return object.NewError(err.Error())
	}
	pbAfter, cont := this.After(action, pb, vnic)
	if !cont {

	}
	if pbAfter != nil {
		return object.New(nil, &l8web.L8Empty{})
	}

	return object.New(nil, &l8web.L8Empty{})
}
