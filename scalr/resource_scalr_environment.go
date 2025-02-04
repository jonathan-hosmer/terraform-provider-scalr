package scalr

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scalr/go-scalr"
)

func resourceScalrEnvironment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalrEnvironmentCreate,
		ReadContext:   resourceScalrEnvironmentRead,
		DeleteContext: resourceScalrEnvironmentDelete,
		UpdateContext: resourceScalrEnvironmentUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cost_estimation_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_by": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"username": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"email": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"full_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"account_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				DefaultFunc: scalrAccountIDDefaultFunc,
				ForceNew:    true,
			},
			"cloud_credentials": {
				Type:       schema.TypeList,
				Computed:   true,
				Optional:   true,
				Elem:       &schema.Schema{Type: schema.TypeString},
				Deprecated: "The attribute `cloud_credentials` is deprecated. Use `default_provider_configurations` instead",
			},
			"policy_groups": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"default_provider_configurations": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"tag_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func parseCloudCredentialDefinitions(d *schema.ResourceData) ([]*scalr.CloudCredential, error) {
	var cloudCredentials []*scalr.CloudCredential

	cloudCredIds := d.Get("cloud_credentials").([]interface{})
	err := ValidateIDsDefinitions(cloudCredIds)
	if err != nil {
		return nil, fmt.Errorf("Got error during parsing cloud credentials: %s", err.Error())
	}

	for _, cloudCredID := range cloudCredIds {
		cloudCredentials = append(cloudCredentials, &scalr.CloudCredential{ID: cloudCredID.(string)})
	}

	return cloudCredentials, nil
}

func parsePolicyGroupDefinitions(d *schema.ResourceData) ([]*scalr.PolicyGroup, error) {
	var policyGroups []*scalr.PolicyGroup

	policyGroupIds := d.Get("policy_groups").([]interface{})
	err := ValidateIDsDefinitions(policyGroupIds)
	if err != nil {
		return nil, fmt.Errorf("Got error during parsing policy groups: %s", err.Error())
	}

	for _, policyGroupID := range policyGroupIds {
		policyGroups = append(policyGroups, &scalr.PolicyGroup{ID: policyGroupID.(string)})
	}

	return policyGroups, nil
}

func resourceScalrEnvironmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)

	name := d.Get("name").(string)
	accountID := d.Get("account_id").(string)
	cloudCredentials, err := parseCloudCredentialDefinitions(d)
	if err != nil {
		return diag.FromErr(err)
	}
	policyGroups, err := parsePolicyGroupDefinitions(d)
	if err != nil {
		return diag.FromErr(err)
	}

	options := scalr.EnvironmentCreateOptions{
		Name:                  scalr.String(name),
		CostEstimationEnabled: scalr.Bool(d.Get("cost_estimation_enabled").(bool)),
		Account:               &scalr.Account{ID: accountID},
		CloudCredentials:      cloudCredentials,
		PolicyGroups:          policyGroups,
	}
	if defaultProviderConfigurationsI, ok := d.GetOk("default_provider_configurations"); ok {
		defaultProviderConfigurations := defaultProviderConfigurationsI.(*schema.Set).List()
		pcfgValues := make([]*scalr.ProviderConfiguration, 0)
		for _, pcfg := range defaultProviderConfigurations {
			pcfgValues = append(pcfgValues, &scalr.ProviderConfiguration{ID: pcfg.(string)})
		}
		options.DefaultProviderConfigurations = pcfgValues

	}
	if tagIDs, ok := d.GetOk("tag_ids"); ok {
		tagIDsList := tagIDs.(*schema.Set).List()
		tags := make([]*scalr.Tag, len(tagIDsList))
		for i, id := range tagIDsList {
			tags[i] = &scalr.Tag{ID: id.(string)}
		}
		options.Tags = tags
	}

	log.Printf("[DEBUG] Create Environment %s for account: %s", name, accountID)
	environment, err := scalrClient.Environments.Create(ctx, options)
	if err != nil {
		return diag.Errorf(
			"Error creating Environment %s for account %s: %v", name, accountID, err)
	}
	d.SetId(environment.ID)
	return resourceScalrEnvironmentRead(ctx, d, meta)
}

func resourceScalrEnvironmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)

	environmentID := d.Id()

	log.Printf("[DEBUG] Read configuration of environment: %s", environmentID)
	environment, err := scalrClient.Environments.Read(ctx, environmentID)
	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound) {
			// If the resource isn't available, the function should set the ID
			// to an empty string so Terraform "destroys" the resource in state.
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error reading environment %s: %v", environmentID, err)
	}

	// Update the configuration.
	_ = d.Set("name", environment.Name)
	_ = d.Set("account_id", environment.Account.ID)
	_ = d.Set("cost_estimation_enabled", environment.CostEstimationEnabled)
	_ = d.Set("status", environment.Status)

	defaultProviderConfigurations := make([]string, 0)
	for _, providerConfiguration := range environment.DefaultProviderConfigurations {
		defaultProviderConfigurations = append(defaultProviderConfigurations, providerConfiguration.ID)
	}
	_ = d.Set("default_provider_configurations", defaultProviderConfigurations)

	var createdBy []interface{}
	if environment.CreatedBy != nil {
		createdBy = append(createdBy, map[string]interface{}{
			"username":  environment.CreatedBy.Username,
			"email":     environment.CreatedBy.Email,
			"full_name": environment.CreatedBy.FullName,
		})
	}
	_ = d.Set("created_by", createdBy)

	cloudCredentials := make([]string, 0)
	if environment.CloudCredentials != nil {
		for _, creds := range environment.CloudCredentials {
			cloudCredentials = append(cloudCredentials, creds.ID)
		}
	}
	_ = d.Set("cloud_credentials", cloudCredentials)

	policyGroups := make([]string, 0)
	if environment.PolicyGroups != nil {
		for _, group := range environment.PolicyGroups {
			policyGroups = append(policyGroups, group.ID)
		}
	}
	_ = d.Set("policy_groups", policyGroups)

	var tagIDs []string
	if len(environment.Tags) != 0 {
		for _, tag := range environment.Tags {
			tagIDs = append(tagIDs, tag.ID)
		}
	}
	_ = d.Set("tag_ids", tagIDs)

	return nil
}

func resourceScalrEnvironmentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)

	var err error
	cloudCredentials, err := parseCloudCredentialDefinitions(d)
	if err != nil {
		return diag.FromErr(err)
	}
	policyGroups, err := parsePolicyGroupDefinitions(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Create a new options struct.
	options := scalr.EnvironmentUpdateOptions{
		Name:                  scalr.String(d.Get("name").(string)),
		CostEstimationEnabled: scalr.Bool(d.Get("cost_estimation_enabled").(bool)),
		CloudCredentials:      cloudCredentials,
		PolicyGroups:          policyGroups,
	}

	if defaultProviderConfigurationsI, ok := d.GetOk("default_provider_configurations"); ok {
		defaultProviderConfigurations := defaultProviderConfigurationsI.(*schema.Set).List()
		pcfgValues := make([]*scalr.ProviderConfiguration, 0)
		for _, pcfg := range defaultProviderConfigurations {
			pcfgValues = append(pcfgValues, &scalr.ProviderConfiguration{ID: pcfg.(string)})
		}
		options.DefaultProviderConfigurations = pcfgValues
	} else {
		options.DefaultProviderConfigurations = make([]*scalr.ProviderConfiguration, 0)
	}

	log.Printf("[DEBUG] Update environment: %s", d.Id())
	_, err = scalrClient.Environments.Update(ctx, d.Id(), options)
	if err != nil {
		return diag.Errorf("Error updating environment %s: %v", d.Id(), err)
	}

	if d.HasChange("tag_ids") {
		oldTags, newTags := d.GetChange("tag_ids")
		oldSet := oldTags.(*schema.Set)
		newSet := newTags.(*schema.Set)
		tagsToAdd := InterfaceArrToTagRelationArr(newSet.Difference(oldSet).List())
		tagsToDelete := InterfaceArrToTagRelationArr(oldSet.Difference(newSet).List())

		if len(tagsToAdd) > 0 {
			err := scalrClient.EnvironmentTags.Add(ctx, d.Id(), tagsToAdd)
			if err != nil {
				return diag.Errorf(
					"Error adding tags to environment %s: %v", d.Id(), err)
			}
		}

		if len(tagsToDelete) > 0 {
			err := scalrClient.EnvironmentTags.Delete(ctx, d.Id(), tagsToDelete)
			if err != nil {
				return diag.Errorf(
					"Error deleting tags from environment %s: %v", d.Id(), err)
			}
		}
	}

	return resourceScalrEnvironmentRead(ctx, d, meta)
}

func resourceScalrEnvironmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)
	environmentID := d.Id()

	log.Printf("[DEBUG] Delete environment %s", environmentID)
	err := scalrClient.Environments.Delete(ctx, d.Id())
	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound) {
			return nil
		}
		return diag.Errorf(
			"Error deleting environment %s: %v", environmentID, err)
	}

	return nil
}
