package scalr

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	scalr "github.com/scalr/go-scalr"
)

func resourceScalrIamUserMembership() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalrIamUserMembershipCreate,
		Read:   resourceScalrIamUserMembershipRead,
		Delete: resourceScalrIamUserMembershipDelete,
		Importer: &schema.ResourceImporter{
			State: resourceScalrIamUserMembershipImport,
		},

		Schema: map[string]*schema.Schema{
			"user_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"team_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceScalrIamUserMembershipImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	scalrClient := meta.(*scalr.Client)

	id := d.Id()

	user, team, err := getLinkedUserTeam(id, scalrClient)
	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound{}) {
			return nil, fmt.Errorf("iam user membership %s not found", id)
		}
		return nil, fmt.Errorf("error retrieving iam user membership %s: %v", id, err)
	}

	d.Set("user_id", user.ID)
	d.Set("team_id", team.ID)

	return []*schema.ResourceData{d}, nil
}

func resourceScalrIamUserMembershipCreate(d *schema.ResourceData, meta interface{}) error {
	scalrClient := meta.(*scalr.Client)

	userID := d.Get("user_id").(string)
	teamID := d.Get("team_id").(string)
	id := packIamUserMembershipID(userID, teamID)

	team, err := scalrClient.Teams.Read(ctx, teamID)
	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound{}) {
			return fmt.Errorf("team %s not found", teamID)
		}
		return fmt.Errorf("error creating iam user membership %s: %v", id, err)
	}

	// existing users of the team plus the new one
	users := append(team.Users, &scalr.User{ID: userID})

	opts := scalr.TeamUpdateOptions{Users: users}
	_, err = scalrClient.Teams.Update(ctx, teamID, opts)
	if err != nil {
		return fmt.Errorf("error creating iam user membership %s: %v", id, err)
	}

	d.SetId(id)
	return resourceScalrIamUserMembershipRead(d, meta)
}

func resourceScalrIamUserMembershipRead(d *schema.ResourceData, meta interface{}) error {
	scalrClient := meta.(*scalr.Client)

	id := d.Id()

	user, team, err := getLinkedUserTeam(id, scalrClient)
	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound{}) {
			log.Printf("[DEBUG] Iam user membership %s not found", id)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error retrieving iam user membership %s: %v", id, err)
	}

	d.Set("user_id", user.ID)
	d.Set("team_id", team.ID)

	return nil
}

func resourceScalrIamUserMembershipDelete(d *schema.ResourceData, meta interface{}) error {
	scalrClient := meta.(*scalr.Client)

	id := d.Id()

	user, team, err := getLinkedUserTeam(id, scalrClient)
	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound{}) {
			log.Printf("[DEBUG] Iam user membership %s not found", id)
			return nil
		}
		return fmt.Errorf("error deleting iam user membership %s: %v", id, err)
	}

	// existing users of the team that will remain linked
	var users []*scalr.User
	for _, u := range team.Users {
		if u.ID != user.ID {
			users = append(users, u)
		}
	}

	opts := scalr.TeamUpdateOptions{Users: users}
	_, err = scalrClient.Teams.Update(ctx, team.ID, opts)
	if err != nil {
		return fmt.Errorf("error deleting iam user membership %s: %v", id, err)
	}

	return nil
}

// getLinkedUserTeam verifies existence of the membership
// and returns associated user and team.
func getLinkedUserTeam(id string, scalrClient *scalr.Client) (
	user *scalr.User, team *scalr.Team, err error,
) {
	userID, teamID, err := unpackIamUserMembershipID(id)
	if err != nil {
		return
	}

	team, err = scalrClient.Teams.Read(ctx, teamID)
	if err != nil {
		return
	}

	for _, u := range team.Users {
		if u.ID == userID {
			user = u
			break
		}
	}
	if user == nil {
		return nil, nil, scalr.ErrResourceNotFound{}
	}

	return
}

func packIamUserMembershipID(userID, teamID string) string {
	return userID + "/" + teamID
}

func unpackIamUserMembershipID(id string) (userID, teamID string, err error) {
	if s := strings.SplitN(id, "/", 2); len(s) == 2 {
		return s[0], s[1], nil
	}
	return "", "", fmt.Errorf(
		"invalid iam user membership ID format: %s (expected <user_id>/<team_id>", id,
	)
}
