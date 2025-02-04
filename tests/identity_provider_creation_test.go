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
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"       // nolint
	. "github.com/onsi/gomega"       // nolint
	. "github.com/onsi/gomega/ghttp" // nolint

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Identity provider creation", func() {
	var ctx context.Context
	var server *Server
	var ca string
	var token string

	BeforeEach(func() {
		// Create a contet:
		ctx = context.Background()

		// Create an access token:
		token = MakeTokenString("Bearer", 10*time.Minute)

		// Start the server:
		server, ca = MakeTCPTLSServer()
	})

	AfterEach(func() {
		// Stop the server:
		server.Close()

		// Remove the server CA file:
		err := os.Remove(ca)
		Expect(err).ToNot(HaveOccurred())
	})

	When("There is no identity provider yet", func() {
		BeforeEach(func() {
			// Prepare the server:
			server.AppendHandlers(
				// First thing the provider will do is check if the identity
				// provider exists, and to do so it will fetch all the identity
				// providers of the cluster:
				CombineHandlers(
					VerifyRequest(
						http.MethodGet,
						"/api/clusters_mgmt/v1/clusters/123/identity_providers",
					),
					RespondWithJSON(
						http.StatusOK,
						`{
						  "page": 1,
						  "size": 0,
						  "total": 0,
						  "items": []
						}`,
					),
				),

				// Then it will retrieve the cluster to check that it is ready:
				CombineHandlers(
					VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/123"),
					RespondWithJSON(
						http.StatusOK,
						`{
						  "id": "123",
						  "name": "my-cluster",
						  "state": "ready"
						}`,
					),
				),
			)
		})

		It("Can create a 'htpasswd' identity provider", func() {
			// Prepare the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyRequest(
						http.MethodPost,
						"/api/clusters_mgmt/v1/clusters/123/identity_providers",
					),
					RespondWithJSON(
						http.StatusOK,
						`{
					  "id": "456",
					  "name": "my-ip",
					  "htpasswd": {
					    "user": "my-user"
					  }
					}`,
					),
				),
			)

			// Run the apply command:
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

					provider "ocm" {
					  url         = "{{ .URL }}"
					  token       = "{{ .Token }}"
					  trusted_cas = file("{{ .CA }}")
					}

					resource "ocm_identity_provider" "my_ip" {
					  cluster_id = "123"
					  name       = "my-ip"
					  htpasswd {
					    user     = "my-user"
					    password = "my-password"
					  }
					}
					`,
					"URL", server.URL(),
					"Token", token,
					"CA", ca,
				).
				Args(
					"apply",
					"-auto-approve",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
		})

		It("Can create an LDAP identity provider", func() {
			// Prepare the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyRequest(
						http.MethodPost,
						"/api/clusters_mgmt/v1/clusters/123/identity_providers",
					),
					RespondWithJSON(
						http.StatusOK,
						`{
						  "id": "456",
						  "name": "my-ip",
						  "ldap": {
						  }
						}`,
					),
				),
			)

			// Run the apply command:
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

					provider "ocm" {
					  url         = "{{ .URL }}"
					  token       = "{{ .Token }}"
					  trusted_cas = file("{{ .CA }}")
					}

					resource "ocm_identity_provider" "my_ip" {
					  cluster_id    = "123"
					  name          = "my-ip"
					  ldap {
					    bind_dn       = "my-bind-dn"
					    bind_password = "my-bind-password"
					    url           = "https://my-server.com"
					    attributes {
					      id                 = ["my-id"]
					      email              = ["my-email"]
					      name               = ["my-name"]
					      preferred_username = ["my-preferred-username"]
					    }
					  }
					}
					`,
					"URL", server.URL(),
					"Token", token,
					"CA", ca,
				).
				Args(
					"apply",
					"-auto-approve",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
		})
	})
})
