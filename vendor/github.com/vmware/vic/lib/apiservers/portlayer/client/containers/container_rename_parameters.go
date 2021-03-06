package containers

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"

	strfmt "github.com/go-openapi/strfmt"
)

// NewContainerRenameParams creates a new ContainerRenameParams object
// with the default values initialized.
func NewContainerRenameParams() *ContainerRenameParams {
	var ()
	return &ContainerRenameParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewContainerRenameParamsWithTimeout creates a new ContainerRenameParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewContainerRenameParamsWithTimeout(timeout time.Duration) *ContainerRenameParams {
	var ()
	return &ContainerRenameParams{

		timeout: timeout,
	}
}

// NewContainerRenameParamsWithContext creates a new ContainerRenameParams object
// with the default values initialized, and the ability to set a context for a request
func NewContainerRenameParamsWithContext(ctx context.Context) *ContainerRenameParams {
	var ()
	return &ContainerRenameParams{

		Context: ctx,
	}
}

// NewContainerRenameParamsWithHTTPClient creates a new ContainerRenameParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewContainerRenameParamsWithHTTPClient(client *http.Client) *ContainerRenameParams {
	var ()
	return &ContainerRenameParams{
		HTTPClient: client,
	}
}

/*ContainerRenameParams contains all the parameters to send to the API endpoint
for the container rename operation typically these are written to a http.Request
*/
type ContainerRenameParams struct {

	/*Handle*/
	Handle string
	/*Name
	  New name for the container

	*/
	Name string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the container rename params
func (o *ContainerRenameParams) WithTimeout(timeout time.Duration) *ContainerRenameParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the container rename params
func (o *ContainerRenameParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the container rename params
func (o *ContainerRenameParams) WithContext(ctx context.Context) *ContainerRenameParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the container rename params
func (o *ContainerRenameParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the container rename params
func (o *ContainerRenameParams) WithHTTPClient(client *http.Client) *ContainerRenameParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the container rename params
func (o *ContainerRenameParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithHandle adds the handle to the container rename params
func (o *ContainerRenameParams) WithHandle(handle string) *ContainerRenameParams {
	o.SetHandle(handle)
	return o
}

// SetHandle adds the handle to the container rename params
func (o *ContainerRenameParams) SetHandle(handle string) {
	o.Handle = handle
}

// WithName adds the name to the container rename params
func (o *ContainerRenameParams) WithName(name string) *ContainerRenameParams {
	o.SetName(name)
	return o
}

// SetName adds the name to the container rename params
func (o *ContainerRenameParams) SetName(name string) {
	o.Name = name
}

// WriteToRequest writes these params to a swagger request
func (o *ContainerRenameParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	r.SetTimeout(o.timeout)
	var res []error

	// path param handle
	if err := r.SetPathParam("handle", o.Handle); err != nil {
		return err
	}

	// query param name
	qrName := o.Name
	qName := qrName
	if qName != "" {
		if err := r.SetQueryParam("name", qName); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
