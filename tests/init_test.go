/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"context"

	. "github.com/onsi/ginkgo" // nolint
	. "github.com/onsi/gomega" // nolint
)

var _ = Describe("Init", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("Downloads and installs the provider", func() {
		result := NewCommand().
			File(
				"main.tf", `
				terraform {
				  required_providers {
				    ocm = {
				      source  = "localhost/redhat/ocm"
				    }
				  }
				}
				`,
			).
			Args("validate").
			Run(ctx)
		Expect(result.ExitCode()).To(BeZero())
	})
})
