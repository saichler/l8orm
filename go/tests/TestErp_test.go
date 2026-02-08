package tests

import (
	"fmt"
	"github.com/saichler/l8erp/go/erp/fin/budgetlines"
	"github.com/saichler/l8erp/go/types/erp"
	"github.com/saichler/l8erp/go/types/fin"
	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"testing"
	"time"
)

func TestBudgetLineService(t *testing.T) {
	time.Sleep(time.Second * 2)
	db := openDBConection()
	defer cleanup(db)
	nic := topo.VnicByVnetNum(1, 2)

	budgetlines.Activate("postgres", "erp", nic)
	bs, ok := budgetlines.BudgetLines(nic)
	if !ok {
		panic("not ok")
	}

	bl := &fin.BudgetLine{}
	bl.BudgetId = "id"
	bl.ActualAmount = &erp.Money{}
	bl.ActualAmount.Amount = 100
	bl.ActualAmount.CurrencyCode = "USD"

	el := object.New(nil, bl)

	resp := bs.Post(el, nic)
	if resp.Error() != nil {
		panic(resp.Error().Error())
	}
	fmt.Println(resp.Element())
}

func TestConvertTo(t *testing.T) {
	bl := &fin.BudgetLine{}
	bl.BudgetId = "id"
	bl.ActualAmount = &erp.Money{}
	bl.ActualAmount.Amount = 100
	bl.ActualAmount.CurrencyCode = "USD"

	nic := topo.VnicByVnetNum(1, 2)

	resp := convert.ConvertTo(ifs.POST, object.New(nil, bl), nic.Resources())
	if resp.Error() != nil {
		panic(resp.Error().Error())
	}
	data := resp.Element().(*l8orms.L8OrmRData)
	if len(data.Tables) < 2 {
		t.Fail()
		fmt.Println("Expected 2")
		return
	}
}
