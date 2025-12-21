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
package convert

import (
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8types/go/ifs"
)

const (
	ServiceName = "Convert"
)

type ConvertService struct {
}

func (this *ConvertService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Registry().Register(&l8orms.L8OrmRData{})
	return nil
}

func (this *ConvertService) DeActivate() error {
	return nil
}

func (this *ConvertService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return ConvertTo(ifs.POST, pb, vnic.Resources())
}
func (this *ConvertService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ConvertService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return ConvertTo(ifs.PATCH, pb, vnic.Resources())
}
func (this *ConvertService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ConvertService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return ConvertFrom(pb, nil, vnic.Resources())
}
func (this *ConvertService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ConvertService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

func (this *ConvertService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *ConvertService) WebService() ifs.IWebService {
	return nil
}
