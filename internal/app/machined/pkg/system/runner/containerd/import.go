/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package containerd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	multierror "github.com/hashicorp/go-multierror"

	"github.com/talos-systems/talos/internal/app/machined/pkg/system/conditions"
	"github.com/talos-systems/talos/pkg/constants"
)

// ImportRequest represents an image import request.
type ImportRequest struct {
	Path    string
	Options []containerd.ImportOpt
}

// Importer implements image import
type Importer struct {
	namespace string
	options   importerOptions
}

type importerOptions struct {
	containerdAddress string
}

// ImporterOption configures containerd Inspector
type ImporterOption func(*importerOptions)

// WithContainerdAddress configures containerd address to use
func WithContainerdAddress(address string) ImporterOption {
	return func(o *importerOptions) {
		o.containerdAddress = address
	}
}

// NewImporter builds new Importer
func NewImporter(namespace string, options ...ImporterOption) *Importer {
	importer := &Importer{
		namespace: namespace,
		options: importerOptions{
			containerdAddress: constants.ContainerdAddress,
		},
	}

	for _, opt := range options {
		opt(&importer.options)
	}

	return importer
}

// Import imports the images specified by the import requests.
func (i *Importer) Import(reqs ...*ImportRequest) (err error) {
	err = conditions.WaitForFileToExist(i.options.containerdAddress).Wait(context.Background())
	if err != nil {
		return err
	}

	ctx := namespaces.WithNamespace(context.Background(), i.namespace)

	client, err := containerd.New(i.options.containerdAddress)
	if err != nil {
		return err
	}
	// nolint: errcheck
	defer client.Close()

	errCh := make(chan error)

	var result *multierror.Error

	for _, req := range reqs {
		go func(errCh chan<- error, r *ImportRequest) {
			errCh <- func() error {
				tarball, err := os.Open(r.Path)
				if err != nil {
					return fmt.Errorf("error opening %s: %w", r.Path, err)
				}

				imgs, err := client.Import(ctx, tarball, r.Options...)
				if err != nil {
					return fmt.Errorf("error importing %s: %w", r.Path, err)
				}
				if err = tarball.Close(); err != nil {
					return fmt.Errorf("error closing %s: %w", r.Path, err)
				}

				for _, img := range imgs {
					image := containerd.NewImage(client, img)
					log.Printf("unpacking %s (%s)\n", img.Name, img.Target.Digest)
					err = image.Unpack(ctx, containerd.DefaultSnapshotter)
					if err != nil {
						return fmt.Errorf("error unpacking %s: %w", img.Name, err)
					}
				}

				return nil
			}()
		}(errCh, req)
	}

	for range reqs {
		result = multierror.Append(result, <-errCh)
	}

	return result.ErrorOrNil()
}

// Import imports the images specified by the import requests.
func Import(namespace string, reqs ...*ImportRequest) error {
	return NewImporter(namespace).Import(reqs...)
}
