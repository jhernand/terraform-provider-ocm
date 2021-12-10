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

package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type ClusterResourceType struct {
}

type ClusterResource struct {
	logger     logging.Logger
	collection *cmv1.ClustersClient
}

func (t *ClusterResourceType) GetSchema(ctx context.Context) (result tfsdk.Schema,
	diags diag.Diagnostics) {
	result = tfsdk.Schema{
		Description: "OpenShift managed cluster.",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Description: "Unique identifier of the cluster.",
				Type:        types.StringType,
				Computed:    true,
			},
			"name": {
				Description: "Name of the cluster.",
				Type:        types.StringType,
				Required:    true,
			},
			"cloud_provider": {
				Description: "Cloud provider identifier, for example 'aws'.",
				Type:        types.StringType,
				Required:    true,
			},
			"cloud_region": {
				Description: "Cloud region identifier, for example 'us-east-1'.",
				Type:        types.StringType,
				Required:    true,
			},
			"multi_az": {
				Description: "Indicates if the cluster should be deployed to " +
					"multiple availability zones. Default value is 'false'.",
				Type:     types.BoolType,
				Optional: true,
				Computed: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					tfsdk.RequiresReplace(),
				},
			},
			"properties": {
				Description: "User defined properties.",
				Type: types.MapType{
					ElemType: types.StringType,
				},
				Optional: true,
				Computed: true,
			},
			"api_url": {
				Description: "URL of the API server.",
				Type:        types.StringType,
				Computed:    true,
			},
			"console_url": {
				Description: "URL of the console.",
				Type:        types.StringType,
				Computed:    true,
			},
			"compute_nodes": {
				Description: "Number of compute nodes of the cluster.",
				Type:        types.Int64Type,
				Optional:    true,
				Computed:    true,
			},
			"compute_machine_type": {
				Description: "Identifier of the machine type used by the compute nodes, " +
					"for example `r5.xlarge`. Use the `ocm_machine_types` data " +
					"source to find the possible values.",
				Type:     types.StringType,
				Optional: true,
				Computed: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					tfsdk.RequiresReplace(),
				},
			},
			"ccs_enabled": {
				Description: "Enables customer cloud subscription.",
				Type:        types.BoolType,
				Optional:    true,
				Computed:    true,
			},
			"aws_account_id": {
				Description: "Identifier of the AWS account.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"aws_access_key_id": {
				Description: "Identifier of the AWS access key.",
				Type:        types.StringType,
				Optional:    true,
				Sensitive:   true,
			},
			"aws_secret_access_key": {
				Description: "AWS access key.",
				Type:        types.StringType,
				Optional:    true,
				Sensitive:   true,
			},
			"sts_role": {
				Description: "Name of the AWS role used by the cluster.",
				Type:        types.StringType,
				Optional:    true,
			},
			"sts_support_role": {
				Description: "Name of the AWS role used for support activities.",
				Type:        types.StringType,
				Optional:    true,
			},
			"sts_machine_role": {
				Description: "Name of the AWS role used by the machine " +
					"API controller.",
				Type:     types.StringType,
				Optional: true,
			},
			"sts_cloud_role": {
				Description: "Name of the AWS role used by the cloud " +
					"credentials operator.",
				Type:     types.StringType,
				Optional: true,
			},
			"sts_registry_role": {
				Description: "Name of the AWS role used by image " +
					"registry operator.",
				Type:     types.StringType,
				Optional: true,
			},
			"sts_ingress_role": {
				Description: "Name of the AWS role used by image " +
					"ingress operator.",
				Type:     types.StringType,
				Optional: true,
			},
			"sts_csi_role": {
				Description: "Name of the AWS role used by CSI " +
					"drivers operator.",
				Type:     types.StringType,
				Optional: true,
			},
			"sts_control_role": {
				Description: "Name of the AWS role used by control plane nodes.",
				Type:        types.StringType,
				Optional:    true,
			},
			"sts_worker_role": {
				Description: "Name of the AWS role used by worker nodes.",
				Type:        types.StringType,
				Optional:    true,
			},
			"machine_cidr": {
				Description: "Block of IP addresses for nodes.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"service_cidr": {
				Description: "Block of IP addresses for services.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"pod_cidr": {
				Description: "Block of IP addresses for pods.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"host_prefix": {
				Description: "Length of the prefix of the subnet assigned to each node.",
				Type:        types.Int64Type,
				Optional:    true,
				Computed:    true,
			},
			"version": {
				Description: "Identifier of the version of OpenShift, for example 'openshift-v4.1.0'.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"state": {
				Description: "State of the cluster.",
				Type:        types.StringType,
				Computed:    true,
			},
			"wait": {
				Description: "Wait till the cluster is ready.",
				Type:        types.BoolType,
				Optional:    true,
			},
		},
	}
	return
}

func (t *ClusterResourceType) NewResource(ctx context.Context,
	p tfsdk.Provider) (result tfsdk.Resource, diags diag.Diagnostics) {
	// Cast the provider interface to the specific implementation:
	parent := p.(*Provider)

	// Get the collection of clusters:
	collection := parent.connection.ClustersMgmt().V1().Clusters()

	// Create the resource:
	result = &ClusterResource{
		logger:     parent.logger,
		collection: collection,
	}

	return
}

func (r *ClusterResource) Create(ctx context.Context,
	request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	// Get the plan:
	state := &ClusterState{}
	diags := request.Plan.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Create the cluster:
	builder := cmv1.NewCluster()
	builder.Name(state.Name.Value)
	builder.CloudProvider(cmv1.NewCloudProvider().ID(state.CloudProvider.Value))
	builder.Region(cmv1.NewCloudRegion().ID(state.CloudRegion.Value))
	if !state.MultiAZ.Unknown && !state.MultiAZ.Null {
		builder.MultiAZ(state.MultiAZ.Value)
	}
	if !state.Properties.Unknown && !state.Properties.Null {
		properties := map[string]string{}
		for k, v := range state.Properties.Elems {
			properties[k] = v.(types.String).Value
		}
		builder.Properties(properties)
	}
	nodes := cmv1.NewClusterNodes()
	if !state.ComputeNodes.Unknown && !state.ComputeNodes.Null {
		nodes.Compute(int(state.ComputeNodes.Value))
	}
	if !state.ComputeMachineType.Unknown && !state.ComputeMachineType.Null {
		nodes.ComputeMachineType(
			cmv1.NewMachineType().ID(state.ComputeMachineType.Value),
		)
	}
	if !nodes.Empty() {
		builder.Nodes(nodes)
	}
	ccs := cmv1.NewCCS()
	if !state.CCSEnabled.Unknown && !state.CCSEnabled.Null {
		ccs.Enabled(state.CCSEnabled.Value)
	}
	if !ccs.Empty() {
		builder.CCS(ccs)
	}
	aws := cmv1.NewAWS()
	if !state.AWSAccountID.Unknown && !state.AWSAccountID.Null {
		aws.AccountID(state.AWSAccountID.Value)
	}
	if !state.AWSAccessKeyID.Unknown && !state.AWSAccessKeyID.Null {
		aws.AccessKeyID(state.AWSAccessKeyID.Value)
	}
	if !state.AWSSecretAccessKey.Unknown && !state.AWSSecretAccessKey.Null {
		aws.SecretAccessKey(state.AWSSecretAccessKey.Value)
	}
	awsSTS := cmv1.NewSTS()
	if !state.STSRole.Unknown && !state.STSRole.Null {
		awsSTS.RoleARN(state.STSRole.Value)
	}
	if !state.STSSupportRole.Unknown && !state.STSSupportRole.Null {
		awsSTS.SupportRoleARN(state.STSSupportRole.Value)
	}
	var awsSTSOperatorRoles []*cmv1.OperatorIAMRoleBuilder
	if !state.STSMachineRole.Unknown && !state.STSMachineRole.Null {
		awsSTSOperatorRoles = append(
			awsSTSOperatorRoles,
			cmv1.NewOperatorIAMRole().
				Namespace("openshift-machine-api").
				Name("aws-cloud-credentials").
				RoleARN(state.STSMachineRole.Value),
		)
	}
	if !state.STSCloudRole.Unknown && !state.STSCloudRole.Null {
		awsSTSOperatorRoles = append(
			awsSTSOperatorRoles,
			cmv1.NewOperatorIAMRole().
				Namespace("openshift-cloud-credential-operator").
				Name("cloud-credential-operator-iam-ro-creds").
				RoleARN(state.STSCloudRole.Value),
		)
	}
	if !state.STSRegistryRole.Unknown && !state.STSRegistryRole.Null {
		awsSTSOperatorRoles = append(
			awsSTSOperatorRoles,
			cmv1.NewOperatorIAMRole().
				Namespace("openshift-image-registry").
				Name("installer-cloud-credentials").
				RoleARN(state.STSRegistryRole.Value),
		)
	}
	if !state.STSIngressRole.Unknown && !state.STSIngressRole.Null {
		awsSTSOperatorRoles = append(
			awsSTSOperatorRoles,
			cmv1.NewOperatorIAMRole().
				Namespace("openshift-ingress-operator").
				Name("cloud-credentials").
				RoleARN(state.STSIngressRole.Value),
		)
	}
	if !state.STSCSIRole.Unknown && !state.STSCSIRole.Null {
		awsSTSOperatorRoles = append(
			awsSTSOperatorRoles,
			cmv1.NewOperatorIAMRole().
				Namespace("openshift-cluster-csi-drivers").
				Name("ebs-cloud-credentials").
				RoleARN(state.STSCSIRole.Value),
		)
	}
	if len(awsSTSOperatorRoles) > 0 {
		awsSTS.OperatorIAMRoles(awsSTSOperatorRoles...)
	}
	awsSTSInstanceRoles := cmv1.NewInstanceIAMRoles()
	if !state.STSControlRole.Unknown && !state.STSControlRole.Null {
		awsSTSInstanceRoles.MasterRoleARN(state.STSControlRole.Value)
	}
	if !state.STSWorkerRole.Unknown && !state.STSWorkerRole.Null {
		awsSTSInstanceRoles.WorkerRoleARN(state.STSWorkerRole.Value)
	}
	if !awsSTSInstanceRoles.Empty() {
		awsSTS.InstanceIAMRoles(awsSTSInstanceRoles)
	}
	if !awsSTS.Empty() {
		aws.STS(awsSTS)
	}
	if !aws.Empty() {
		builder.AWS(aws)
	}
	network := cmv1.NewNetwork()
	if !state.MachineCIDR.Unknown && !state.MachineCIDR.Null {
		network.MachineCIDR(state.MachineCIDR.Value)
	}
	if !state.ServiceCIDR.Unknown && !state.ServiceCIDR.Null {
		network.ServiceCIDR(state.ServiceCIDR.Value)
	}
	if !state.PodCIDR.Unknown && !state.PodCIDR.Null {
		network.PodCIDR(state.PodCIDR.Value)
	}
	if !state.HostPrefix.Unknown && !state.HostPrefix.Null {
		network.HostPrefix(int(state.HostPrefix.Value))
	}
	if !network.Empty() {
		builder.Network(network)
	}
	if !state.Version.Unknown && !state.Version.Null {
		builder.Version(cmv1.NewVersion().ID(state.Version.Value))
	}
	object, err := builder.Build()
	if err != nil {
		response.Diagnostics.AddError(
			"Can't build cluster",
			fmt.Sprintf(
				"Can't build cluster with name '%s': %v",
				state.Name.Value, err,
			),
		)
		return
	}
	add, err := r.collection.Add().Body(object).SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't create cluster",
			fmt.Sprintf(
				"Can't create cluster with name '%s': %v",
				state.Name.Value, err,
			),
		)
		return
	}
	object = add.Body()

	// Wait till the cluster is ready unless explicitly disabled:
	wait := state.Wait.Unknown || state.Wait.Null || state.Wait.Value
	ready := object.State() == cmv1.ClusterStateReady
	if wait && !ready {
		pollCtx, cancel := context.WithTimeout(ctx, 1*time.Hour)
		defer cancel()
		_, err := r.collection.Cluster(object.ID()).Poll().
			Interval(30 * time.Second).
			Predicate(func(get *cmv1.ClusterGetResponse) bool {
				object = get.Body()
				return object.State() == cmv1.ClusterStateReady
			}).
			StartContext(pollCtx)
		if err != nil {
			response.Diagnostics.AddError(
				"Can't poll cluster state",
				fmt.Sprintf(
					"Can't poll state of cluster with identifier '%s': %v",
					object.ID(), err,
				),
			)
			return
		}
	}

	// Save the state:
	r.populateState(object, state)
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

func (r *ClusterResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest,
	response *tfsdk.ReadResourceResponse) {
	// Get the current state:
	state := &ClusterState{}
	diags := request.State.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Find the cluster:
	get, err := r.collection.Cluster(state.ID.Value).Get().SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't find cluster",
			fmt.Sprintf(
				"Can't find cluster with identifier '%s': %v",
				state.ID.Value, err,
			),
		)
		return
	}
	object := get.Body()

	// Save the state:
	r.populateState(object, state)
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

func (r *ClusterResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest,
	response *tfsdk.UpdateResourceResponse) {
	var diags diag.Diagnostics

	// Get the state:
	state := &ClusterState{}
	diags = request.State.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Get the plan:
	plan := &ClusterState{}
	diags = request.Plan.Get(ctx, plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Send request to update the cluster:
	builder := cmv1.NewCluster()
	var nodes *cmv1.ClusterNodesBuilder
	compute, ok := shouldPatchInt(state.ComputeNodes, plan.ComputeNodes)
	if ok {
		nodes.Compute(int(compute))
	}
	if !nodes.Empty() {
		builder.Nodes(nodes)
	}
	patch, err := builder.Build()
	if err != nil {
		response.Diagnostics.AddError(
			"Can't build cluster patch",
			fmt.Sprintf(
				"Can't build patch for cluster with identifier '%s': %v",
				state.ID.Value, err,
			),
		)
		return
	}
	update, err := r.collection.Cluster(state.ID.Value).Update().
		Body(patch).
		SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't update cluster",
			fmt.Sprintf(
				"Can't update cluster with identifier '%s': %v",
				state.ID.Value, err,
			),
		)
		return
	}
	object := update.Body()

	// Update the state:
	r.populateState(object, state)
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

func (r *ClusterResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest,
	response *tfsdk.DeleteResourceResponse) {
	// Get the state:
	state := &ClusterState{}
	diags := request.State.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Send the request to delete the cluster:
	resource := r.collection.Cluster(state.ID.Value)
	_, err := resource.Delete().SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't delete cluster",
			fmt.Sprintf(
				"Can't delete cluster with identifier '%s': %v",
				state.ID.Value, err,
			),
		)
		return
	}

	// Wait till the cluster has been effectively deleted:
	if state.Wait.Unknown || state.Wait.Null || state.Wait.Value {
		pollCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		_, err := resource.Poll().
			Interval(30 * time.Second).
			Status(http.StatusNotFound).
			StartContext(pollCtx)
		sdkErr, ok := err.(*errors.Error)
		if ok && sdkErr.Status() == http.StatusNotFound {
			err = nil
		}
		if err != nil {
			response.Diagnostics.AddError(
				"Can't poll cluster deletion",
				fmt.Sprintf(
					"Can't poll deletion of cluster with identifier '%s': %v",
					state.ID.Value, err,
				),
			)
			return
		}
	}

	// Remove the state:
	response.State.RemoveResource(ctx)
}

func (r *ClusterResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest,
	response *tfsdk.ImportResourceStateResponse) {
	// Try to retrieve the object:
	get, err := r.collection.Cluster(request.ID).Get().SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't find cluster",
			fmt.Sprintf(
				"Can't find cluster with identifier '%s': %v",
				request.ID, err,
			),
		)
		return
	}
	object := get.Body()

	// Save the state:
	state := &ClusterState{}
	r.populateState(object, state)
	diags := response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

// populateState copies the data from the API object to the Terraform state.
func (r *ClusterResource) populateState(object *cmv1.Cluster, state *ClusterState) {
	state.ID = types.String{
		Value: object.ID(),
	}
	state.Name = types.String{
		Value: object.Name(),
	}
	state.CloudProvider = types.String{
		Value: object.CloudProvider().ID(),
	}
	state.CloudRegion = types.String{
		Value: object.Region().ID(),
	}
	state.MultiAZ = types.Bool{
		Value: object.MultiAZ(),
	}
	state.Properties = types.Map{
		ElemType: types.StringType,
		Elems:    map[string]attr.Value{},
	}
	for k, v := range object.Properties() {
		state.Properties.Elems[k] = types.String{
			Value: v,
		}
	}
	state.APIURL = types.String{
		Value: object.API().URL(),
	}
	state.ConsoleURL = types.String{
		Value: object.Console().URL(),
	}
	state.ComputeNodes = types.Int64{
		Value: int64(object.Nodes().Compute()),
	}
	state.ComputeMachineType = types.String{
		Value: object.Nodes().ComputeMachineType().ID(),
	}
	state.CCSEnabled = types.Bool{
		Value: object.CCS().Enabled(),
	}
	awsAccountID, ok := object.AWS().GetAccountID()
	if ok {
		state.AWSAccountID = types.String{
			Value: awsAccountID,
		}
	} else {
		state.AWSAccountID = types.String{
			Null: true,
		}
	}
	awsAccessKeyID, ok := object.AWS().GetAccessKeyID()
	if ok {
		state.AWSAccessKeyID = types.String{
			Value: awsAccessKeyID,
		}
	} else {
		state.AWSAccessKeyID = types.String{
			Null: true,
		}
	}
	awsSecretAccessKey, ok := object.AWS().GetSecretAccessKey()
	if ok {
		state.AWSSecretAccessKey = types.String{
			Value: awsSecretAccessKey,
		}
	} else {
		state.AWSSecretAccessKey = types.String{
			Null: true,
		}
	}
	stsRoleARN, ok := object.AWS().STS().GetRoleARN()
	if ok {
		state.STSRole = types.String{
			Value: stsRoleARN,
		}
	} else {
		state.STSRole = types.String{
			Null: true,
		}
	}
	stsSupportRoleARN, ok := object.AWS().STS().GetSupportRoleARN()
	if ok {
		state.STSSupportRole = types.String{
			Value: stsSupportRoleARN,
		}
	} else {
		state.STSSupportRole = types.String{
			Null: true,
		}
	}
	machineCIDR, ok := object.Network().GetMachineCIDR()
	if ok {
		state.MachineCIDR = types.String{
			Value: machineCIDR,
		}
	} else {
		state.MachineCIDR = types.String{
			Null: true,
		}
	}
	serviceCIDR, ok := object.Network().GetServiceCIDR()
	if ok {
		state.ServiceCIDR = types.String{
			Value: serviceCIDR,
		}
	} else {
		state.ServiceCIDR = types.String{
			Null: true,
		}
	}
	podCIDR, ok := object.Network().GetPodCIDR()
	if ok {
		state.PodCIDR = types.String{
			Value: podCIDR,
		}
	} else {
		state.PodCIDR = types.String{
			Null: true,
		}
	}
	hostPrefix, ok := object.Network().GetHostPrefix()
	if ok {
		state.HostPrefix = types.Int64{
			Value: int64(hostPrefix),
		}
	} else {
		state.HostPrefix = types.Int64{
			Null: true,
		}
	}
	version, ok := object.Version().GetID()
	if ok {
		state.Version = types.String{
			Value: version,
		}
	} else {
		state.Version = types.String{
			Null: true,
		}
	}
	state.State = types.String{
		Value: string(object.State()),
	}
}
