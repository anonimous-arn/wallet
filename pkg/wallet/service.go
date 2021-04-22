package wallet

import (
	"sync"
	"io"
	"strings"
	"strconv"
	"os"
	"log"
	"errors"
	"github.com/anonimous-arn/wallet/pkg/types"
	"github.com/google/uuid"
	
)

var ErrPhoneRegistered = errors.New("phone already registered")
var ErrAmountMustBePositive = errors.New("amount must be greater then 0")
var ErrAccountNotFound = errors.New("account not found")
var ErrPaymentNotFound = errors.New("payment not found")
var ErrNotEnoughBalance = errors.New("account balance least then amount")
var ErrFavoriteNotFound = errors.New("favorite payment not found")

type Service struct {
	nextAccountID int64
	accounts      []*types.Account
	payments      []*types.Payment
	favorites     []*types.Favorite
}

func (s *Service) RegisterAccount(phone types.Phone) (*types.Account, error) {
	for _, account := range s.accounts {
		if account.Phone == phone {
			return nil, ErrPhoneRegistered
		}
	}

	s.nextAccountID++
	account := &types.Account{
		ID:      s.nextAccountID,
		Phone:   phone,
		Balance: 0,
	}
	s.accounts = append(s.accounts, account)
	return account, nil
}

func (s *Service) Deposit(accountID int64, amount types.Money) error {
	if amount <= 0 {
		return ErrAmountMustBePositive
	}

	account, err := s.FindAccountByID(accountID)
	if err != nil {
		return err
	}

	account.Balance += amount
	return nil
}

func (s *Service) Pay(accountID int64, amount types.Money, category types.PaymentCategory) (*types.Payment, error) {
	if amount <= 0 {
		return nil, ErrAmountMustBePositive
	}

	account, err := s.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}

	if account.Balance < amount {
		return nil, ErrNotEnoughBalance
	}

	account.Balance -= amount

	paymentID := uuid.New().String()

	payment := &types.Payment{
		ID:        paymentID,
		AccountID: accountID,
		Amount:    amount,
		Category:  category,
		Status:    types.PaymentStatusInProgress,
	}

	s.payments = append(s.payments, payment)

	return payment, nil
}

func (s *Service) FindAccountByID(accountID int64) (*types.Account, error) {
	var account *types.Account

	for _, acc := range s.accounts {
		if acc.ID == accountID {
			account = acc
			break
		}
	}

	if account == nil {
		return nil, ErrAccountNotFound
	}

	return account, nil
}

func (s *Service) FindPaymentByID(paymentID string) (*types.Payment, error) {
	for _, payment := range s.payments {
		if payment.ID == paymentID {
			return payment, nil
		}
	}

	return nil, ErrPaymentNotFound
}

func (s *Service) Reject(paymentID string) error {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return err
	}

	account, err := s.FindAccountByID(payment.AccountID)
	if err != nil {
		return err
	}

	account.Balance += payment.Amount
	payment.Amount = 0
	payment.Status = types.PaymentStatusFail
	return nil
}

func (s *Service) Repeat(paymentID string) (*types.Payment, error) {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return nil, err
	}

	newPayment, err := s.Pay(payment.AccountID, payment.Amount, payment.Category)
	if err != nil {
		return nil, err
	}

	return newPayment, nil
}

func (s *Service) FavoritePayment(paymentID string, name string) (*types.Favorite, error) {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return nil, err
	}

	favorite := &types.Favorite{
		ID:        uuid.New().String(),
		AccountID: payment.AccountID,
		Name:      name,
		Amount:    payment.Amount,
		Category:  payment.Category,
	}

	s.favorites = append(s.favorites, favorite)
	return favorite, nil
}

func (s *Service) PayFromFavorite(favoriteID string) (*types.Payment, error) {
	var targetFavorite *types.Favorite

	for _, favorite := range s.favorites {
		if favorite.ID == favoriteID {
			targetFavorite = favorite
			break
		}
	}

	if targetFavorite == nil {
		return nil, ErrFavoriteNotFound
	}

	payment, err := s.Pay(targetFavorite.AccountID, targetFavorite.Amount, targetFavorite.Category)
	if err != nil {
		return nil, err
	}

	return payment, nil
}
func (s *Service) ExportToFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Print(err)
		}
	} ()

	content := make([]byte, 0)
	for _, account := range s.accounts {
		content = append(content, []byte(strconv.FormatInt(account.ID, 10))...)
		content = append(content, []byte(";")...)
		content = append(content, []byte(account.Phone)...)
		content = append(content, []byte(";")...)
		content = append(content, []byte(strconv.FormatInt(int64(account.Balance), 10))...)
		content = append(content, []byte("|")...)
	}

	_, err = file.Write(content)
	if err != nil {
		log.Print(err)
		return err
	}

	return nil
}
func (s *Service) ImportFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		log.Print(err)
		return err
	}
	defer func() {
		err :=file.Close()
		if err != nil {
			log.Print(err)
		}
	}()

	content := make([]byte, 0)
	buf := make([]byte, 4)
	for {
		read, err := file.Read(buf)
		if err == io.EOF {
			content = append(content, buf[:read]...)
			break
		}

		content = append(content, buf[:read]...)
	}

	log.Print(string(content))
	for _, row := range strings.Split(string(content), "|") {
		col := strings.Split(row, ";")
		if len(col) == 3 {
			s.RegisterAccount(types.Phone(col[1]))
		}
	}

	for _, account := range s.accounts {
		log.Println(account)
	}
	return nil
}
func actionByFile(path, data string) error {
	file, err := os.Create(path)
	if err != nil {
		log.Println(err)
		return err
	}

	defer func() {
		err = file.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	_, err = file.WriteString(data)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (s *Service) ExportAccountHistory(accountID int64) (payments []types.Payment, err error) {
	_, err = s.FindAccountByID(accountID)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for _, payment := range s.payments {
		if payment.AccountID == accountID {
			payments = append(payments, *payment)
		}
	}

	if len(payments) == 0 {
		log.Println("empty payment")
		return nil, ErrPaymentNotFound
	}

	return payments, nil
}

func (s *Service) HistoryToFiles(payments []types.Payment, dir string, records int) error {
	if len(payments) == 0 {
		log.Print(ErrPaymentNotFound)
		return nil
	}

	//log.Printf("payments = %v \n dir = %v \n records = %v", payments, dir, records)

	if len(payments) <= records {
		result := ""
		for _, payment := range payments {
			result += payment.ID + ";"
			result += strconv.Itoa(int(payment.AccountID)) + ";"
			result += strconv.Itoa(int(payment.Amount)) + ";"
			result += string(payment.Category) + ";"
			result += string(payment.Status) + "\n"
		}

		err := actionByFile(dir+"/payments.dump", result)
		if err != nil {
			return err
		}

		return nil
	}

	result := ""
	k := 1
	for i, payment := range payments {
		result += payment.ID + ";"
		result += strconv.Itoa(int(payment.AccountID)) + ";"
		result += strconv.Itoa(int(payment.Amount)) + ";"
		result += string(payment.Category) + ";"
		result += string(payment.Status) + "\n"

		if (i+1)%records == 0 {
			err := actionByFile(dir+"/payments"+strconv.Itoa(k)+".dump", result)
			if err != nil {
				return err
			}
			k++
			result = ""
		}
	}

	if result != "" {
		err := actionByFile(dir+"/payments"+strconv.Itoa(k)+".dump", result)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) SumPayments(goroutines int) types.Money {
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	var summ types.Money = 0
	if goroutines == 0 || goroutines == 1 {
		wg.Add(1)
		go func(payments []*types.Payment) {
			defer wg.Done()
			for _, payment := range payments {
				summ += payment.Amount
			}
		}(s.payments)
	} else {
		from := 0
		count := len(s.payments) / goroutines
		for i := 1; i <= goroutines; i++ {
			wg.Add(1)
			last := len(s.payments) - i*count
			if i == goroutines {
				last = 0
			}
			to := len(s.payments) - last
			go func(payments []*types.Payment) {
				defer wg.Done()
				s := types.Money(0)
				for _, payment := range payments {
					s += payment.Amount
				}
				mu.Lock()
				defer mu.Unlock()
				summ += s
			}(s.payments[from:to])
			from += count
		}
	}

	wg.Wait()

	return summ
}

func (s *Service) FilterPayments(accountID int64, goroutines int) ([]types.Payment, error) {
	filteredPayments := []types.Payment{}
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	if goroutines == 0 || goroutines == 1 {
		wg.Add(1)
		go func(payments []*types.Payment) {
			defer wg.Done()
			for _, payment := range payments {
				if payment.AccountID == accountID {
					filteredPayments = append(filteredPayments, types.Payment{
						ID:        payment.ID,
						AccountID: payment.AccountID,
						Amount:    payment.Amount,
						Category:  payment.Category,
						Status:    payment.Status,
					})
				}
			}
		}(s.payments)
	} else {
		from := 0
		count := len(s.payments) / goroutines
		for i := 1; i <= goroutines; i++ {
			wg.Add(1)
			last := len(s.payments) - i*count
			if i == goroutines {
				last = 0
			}
			to := len(s.payments) - last
			go func(payments []*types.Payment) {
				defer wg.Done()
				separetePayments := []types.Payment{}
				for _, payment := range payments {
					if payment.AccountID == accountID {
						separetePayments = append(separetePayments, types.Payment{
							ID:        payment.ID,
							AccountID: payment.AccountID,
							Amount:    payment.Amount,
							Category:  payment.Category,
							Status:    payment.Status,
						})
					}
				}
				mu.Lock()
				defer mu.Unlock()
				filteredPayments = append(filteredPayments, separetePayments...)
			}(s.payments[from:to])
			from += count
		}
	}

	wg.Wait()

	if len(filteredPayments) == 0 {
		return nil, ErrAccountNotFound
	}

	return filteredPayments, nil
}