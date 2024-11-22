package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 -o gen.go -package main -generate models,client -response-type-suffix=APIResponse https://console.pomerium.app/openapi.yaml

const (
	apiBaseURL = "https://console.pomerium.app"
	apiPath    = "/api/v0"
	policyName = "jit-example"
)

var getClient = memoize(func(ctx context.Context) (*ClientWithResponses, error) {
	var (
		api *ClientWithResponses

		mu             sync.Mutex
		idToken        string
		idTokenExpires time.Time
	)
	addAuthorization := func(ctx context.Context, req *http.Request) error {
		log.Info().Str("method", req.Method).Str("path", req.URL.Path).Msg("api request")

		// ignore the token path
		if req.URL.Path == apiPath+"/token" {
			return nil
		}

		mu.Lock()
		defer mu.Unlock()

		if time.Now().After(idTokenExpires) {
			idTokenResponse, err := api.GetIdTokenWithResponse(ctx, GetIdTokenJSONRequestBody{
				RefreshToken: config.apiUserToken,
			})
			if err != nil {
				return fmt.Errorf("error retrieving id token: %w", err)
			} else if idTokenResponse.StatusCode() != 200 {
				return fmt.Errorf("unexpected status code from retrieving the id token: %d %s", idTokenResponse.StatusCode(), idTokenResponse.Status())
			}
			expiresInSeconds, err := strconv.Atoi(idTokenResponse.JSON200.ExpiresInSeconds)
			if err != nil {
				return fmt.Errorf("unexpected expiresInSeconds in id token response: %w", err)
			}
			idToken = idTokenResponse.JSON200.IdToken
			idTokenExpires = time.Now().Add(time.Duration(expiresInSeconds) * time.Second)
		}

		req.Header.Set("Authorization", "Bearer "+idToken)
		return nil
	}
	var err error
	api, err = NewClientWithResponses(apiBaseURL+apiPath, WithRequestEditorFn(addAuthorization))
	return api, err
})

var getOrCreatePolicyID = memoize(func(ctx context.Context) (string, error) {
	client, err := getClient(ctx)
	if err != nil {
		return "", err
	}

	res, err := client.ListPoliciesWithResponse(ctx, config.organizationID, &ListPoliciesParams{
		NamespaceId: config.clusterID,
	})
	if err != nil {
		return "", err
	} else if res.StatusCode() != 200 {
		return "", fmt.Errorf("unexpected status code from listing policies: %d", res.StatusCode())
	}

	for _, policy := range *res.JSON200 {
		if policy.Name == policyName {
			return policy.Id, nil
		}
	}

	ppl := PolicyProperties_Ppl{}
	_ = ppl.FromPPLRule(PPLRule{})
	createRes, err := client.CreatePolicyWithResponse(ctx, config.organizationID, CreatePolicyJSONRequestBody{
		Description: "Example policy demonstrating Just-In-Time access",
		Name:        policyName,
		NamespaceId: config.clusterID,
		Ppl:         ppl,
	})
	if err != nil {
		return "", err
	} else if createRes.StatusCode() != 201 {
		return "", fmt.Errorf("unexpected status code from creating policy: %d", createRes.StatusCode())
	}
	return (createRes.JSON201.Id), nil
})

func getPolicy(ctx context.Context) (*Policy, error) {
	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	policyID, err := getOrCreatePolicyID(ctx)
	if err != nil {
		return nil, err
	}

	res, err := client.GetPolicyWithResponse(ctx, config.organizationID, policyID)
	if err != nil {
		return nil, err
	} else if res.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code from retrieving policy: %d", res.StatusCode())
	}
	return res.JSON200, nil
}

func updatePolicy(ctx context.Context, policy *Policy) error {
	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	res, err := client.UpdatePolicyWithResponse(ctx, config.organizationID, policy.Id, UpdatePolicyJSONRequestBody{
		Description: policy.Description,
		Enforced:    policy.Enforced,
		Explanation: policy.Explanation,
		Name:        policy.Name,
		NamespaceId: policy.NamespaceId,
		Ppl:         PolicyProperties_Ppl(policy.Ppl),
		Remediation: policy.Remediation,
	})
	if err != nil {
		return err
	} else if res.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code from updating policy: %d", res.StatusCode())
	}
	return nil
}

func memoize[T any](f func(ctx context.Context) (T, error)) func(ctx context.Context) (T, error) {
	var once sync.Once
	var result T
	var err error
	return func(ctx context.Context) (T, error) {
		once.Do(func() {
			result, err = f(ctx)
		})
		return result, err
	}
}
