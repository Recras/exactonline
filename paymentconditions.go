package exactonline

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/Recras/exactonline/httperror"
)

type PaymentCondition struct {
	ID          string `json:",omitempty"`
	Code        string
	Description string
}
type paymentConditions struct {
	D struct {
		Results []PaymentCondition `json:"results"`
	} `json:"d"`
}

var ErrPaymentConditionNotFound = errors.New("PaymentCondition not Found")

const paymentConditionURI = "/api/v1/%d/cashflow/PaymentConditions"

func (cl *Client) FindPaymentConditionByDescription(desc string) (PaymentCondition, error) {
	filt := url.Values{}
	filt.Set("$filter", fmt.Sprintf("Description eq '"+desc+"'"))
	u := fmt.Sprintf(paymentConditionURI+"?%s", cl.Division, filt.Encode())

	resp, err := cl.Client.Get(u)
	if err != nil {
		return PaymentCondition{}, err
	}
	if resp.StatusCode != 200 {
		return PaymentCondition{}, httperror.New(resp)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	pc := paymentConditions{}
	if err := dec.Decode(&pc); err != nil {
		return PaymentCondition{}, err
	}

	if len(pc.D.Results) == 0 {
		return PaymentCondition{}, ErrPaymentConditionNotFound
	}
	return pc.D.Results[0], nil
}
