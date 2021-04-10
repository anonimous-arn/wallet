package wallet

import (
	"github.com/anonimous-arn/wallet/pkg/types"
	"reflect"
	"testing"
	"fmt"
)

type testAccount struct {
	phone		types.Phone
	balance		types.Money
	payments	[]struct{
		amount		types.Money
		category	types.PaymentCategory
	}
}

var defaultTestAccount = testAccount{
	phone:		"+992000000001",
	balance: 	10_000_00,
	payments: 	[]struct{
		amount		types.Money
		category	types.PaymentCategory
	}{
		{amount: 1_000_00, category: "auto"},
	},
}

type testService struct {
	*Service
}

func (s *testService) addAccount(data testAccount) (*types.Account, []*types.Payment, error) {
	account, err := s.RegisterAccount(data.phone)
	if err != nil {
		return nil, nil, fmt.Errorf("can't regist account,  error = %v", err)
	}

	err = s.Deposit(account.ID, data.balance)
	if err != nil {
		return nil, nil, fmt.Errorf("can't deposity account, error = %v", err)
	}

	payments := make([]*types.Payment, len(data.payments))
	for i, payment := range data.payments {
		payments[i], err = s.Pay(account.ID, payment.amount, payment.category)
		if err != nil {
			return nil, nil, fmt.Errorf("can't make payment, error = %v", err)
		}
	}

	return account, payments, nil
}
func TestService_FindAccountByID_success(t *testing.T) {
	svc := &Service{}

	account, _ := svc.RegisterAccount("+992000000001")

	acc, e := svc.FindAccountByID(account.ID)

	if e != nil {
		t.Error(e)
	}

	if !reflect.DeepEqual(account, acc) {
		t.Error("Accounts doesn't match")
	}
}

func TestService_FindAccountByID_notFound(t *testing.T) {
	svc := &Service{}

	_, e := svc.FindAccountByID(123)

	if e != ErrAccountNotFound {
		t.Error(e)
	}
}

func TestService_FindPaymentByID_success(t *testing.T) {
	svc := &Service{}

	account, errorReg := svc.RegisterAccount("+992000000001")

	if errorReg != nil {
		t.Error("error on register account")
	}

	_, er := svc.Pay(account.ID, 1000, "auto")

	if er == ErrAmountMustBePositive {
		t.Error(ErrAmountMustBePositive)
	}

	if er == nil {
		t.Error("error on pay")
	}
}

func TestService_FindPaymentByID_notFound(t *testing.T) {
	svc := &Service{}

	_, err := svc.FindPaymentByID("aaa")

	if err != ErrPaymentNotFound {
		t.Error("payment exist")
	}
}

func TestService_Reject_success(t *testing.T) {
	svc := &Service{}

	account, errAccount := svc.RegisterAccount("992000000001")

	if errAccount == ErrPhoneRegistered {
		t.Error(ErrPhoneRegistered)
	}

	_, errPay := svc.Pay(account.ID, 1000, "auto")

	if errPay != ErrNotEnoughBalance {
		t.Error(ErrNotEnoughBalance)
	}
}

func TestService_Reject_notRejectPaymentNotFound(t *testing.T) {
	svc := &Service{}

	account, errAccount := svc.RegisterAccount("992000000001")

	if errAccount == ErrPhoneRegistered {
		t.Error(ErrPhoneRegistered)
	}

	_, errPay := svc.Pay(account.ID, 1000, "auto")

	if errPay != ErrNotEnoughBalance {
		t.Error(ErrNotEnoughBalance)
	}
}

func newTestService() *testService {
	return &testService{Service: &Service{}}
}
func TestService_Repeat_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	newPayment, nil := s.Repeat(payment.ID)
	if err != nil {
		t.Error(err)
		return
	}

	if payment.ID == newPayment.ID {
		t.Error("repeated payment id not different")
		return
	}

	if payment.AccountID != newPayment.AccountID ||
		payment.Status != newPayment.Status ||
		payment.Category != newPayment.Category ||
		payment.Amount != newPayment.Amount {
		t.Error("some field is not equal the original")
	}
}