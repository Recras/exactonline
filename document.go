package exactonline

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/Recras/exactonline/httperror"
	"github.com/Recras/exactonline/odata2json"
)

type Document struct {
	ID      string `json:",omitempty"`
	Subject string
	Type    int
	Account string `json:",omitempty"`
}
type documents struct {
	D struct {
		Results []Document `json:"results"`
	} `json:"d"`
}

const documentURI = "/api/v1/%d/documents/Documents"

var ErrNoSubject = errors.New("Document has no Subject")
var ErrNoType = errors.New("Document has no Type")
var ErrNoAccount = errors.New("Document has no Account")

func (d *Document) Save(cl *Client) error {
	if cl.Division == 0 {
		return ErrNoDivision
	}
	if d.Subject == "" {
		return ErrNoSubject
	}
	if d.Type == 0 {
		return ErrNoType
	}
	if d.Account == "" {
		return ErrNoAccount
	}

	bs, err := json.Marshal(d)
	if err != nil {
		return err
	}
	bb := bytes.NewBuffer(bs)

	u := fmt.Sprintf(documentURI, cl.Division)
	resp, err := cl.Client.Post(u, "application/json", bb)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return httperror.New(resp)
	}
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	str := map[string]Document{}
	err = dec.Decode(&str)
	if err != nil {
		return err
	}

	*d = str["d"]

	return nil
}

type DocumentAttachment struct {
	ID         string `json:",omitempty"`
	Attachment odata2json.Binary
	Document   string
	FileName   string
}
type documentattachments struct {
	D struct {
		Results []DocumentAttachment `json:"results"`
	} `json:"d"`
}

const documentAttachmentURI = "/api/v1/%d/documents/DocumentAttachments"

var ErrNoFileName = errors.New("DocumentAttachment has no FileName")
var ErrNoDocument = errors.New("DocumentAttachment has no Document")
var ErrNoAttachment = errors.New("DocumentAttachment has no Attachment")

func (d *DocumentAttachment) Save(cl *Client) error {
	if cl.Division == 0 {
		return ErrNoDivision
	}

	if d.FileName == "" {
		return ErrNoFileName
	}
	if d.Document == "" {
		return ErrNoDocument
	}
	if d.Attachment == nil {
		return ErrNoAttachment
	}

	bs, err := json.Marshal(d)
	if err != nil {
		return err
	}
	bb := bytes.NewBuffer(bs)

	u := fmt.Sprintf(documentAttachmentURI, cl.Division)
	resp, err := cl.Client.Post(u, "application/json", bb)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return httperror.New(resp)
	}
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	str := map[string]DocumentAttachment{}
	err = dec.Decode(&str)
	if err != nil {
		return err
	}

	*d = str["d"]

	return nil
}

type DocumentType struct {
	ID          int
	Description string
}
type documentTypes struct {
	D struct {
		Results []DocumentType `json:"results"`
	} `json:"d"`
}

const documentTypeURI = "/api/v1/%d/documents/DocumentTypes"

type ErrDocumentTypeNotFound struct{}

func (e ErrDocumentTypeNotFound) Error() string {
	return fmt.Sprintf("DocumentType not found")
}

func (c *Client) FindDocumentTypeByDescription(d string) (DocumentType, error) {
	if c.Division == 0 {
		return DocumentType{}, ErrNoDivision
	}
	u := fmt.Sprintf(documentTypeURI, c.Division)
	filt := url.Values{}
	filt.Set("$filter", fmt.Sprintf("Description eq '%s'", d))
	resp, err := c.Client.Get(fmt.Sprintf("%s?%s", u, filt.Encode()))
	if err != nil {
		return DocumentType{}, err
	}
	if resp.StatusCode != 200 {
		return DocumentType{}, httperror.New(resp)
	}
	defer resp.Body.Close()

	out := &documentTypes{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return DocumentType{}, err
	}
	if len(out.D.Results) == 0 {
		return DocumentType{}, ErrDocumentTypeNotFound{}
	}
	return out.D.Results[0], nil
}
