package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/urfave/cli/v2"
)

var httpClt = resty.New()

func main() {
	app := &cli.App{
		Name:  "github-app-token",
		Usage: "Generate a JWT token that can be used to authenticate as a GitHub App.",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:    "expiry",
				Aliases: []string{"e"},
				Value:   5 * time.Minute,
				Usage:   "amount of time before the JWT token expires, as a duration (e.g., 15m)",
			},
			&cli.StringFlag{
				Name:    "repo",
				Aliases: []string{"r"},
				Usage:   "the full repository name that the token is scoped for (e.g., fensak-io/github-app-token). Required.",
			},
		},
		Action: func(ctx *cli.Context) error {
			expiry := ctx.Duration("expiry")
			repo := ctx.String("repo")
			if repo == "" {
				return errors.New("--repo is required")
			}
			appID := os.Getenv("GITHUB_APP_ID")
			if appID == "" {
				return errors.New("env var GITHUB_APP_ID is required to be set")
			}
			pemKey := os.Getenv("GITHUB_APP_PRIVATE_KEY")
			if pemKey == "" {
				return errors.New("env var GITHUB_APP_PRIVATE_KEY is required to be set")
			}

			jwt, err := generateAppJWT(appID, []byte(pemKey), expiry)
			if err != nil {
				return err
			}
			instID, err := getInstallationID(jwt, repo)
			if err != nil {
				return err
			}
			token, err := getAccessToken(jwt, instID)
			if err != nil {
				return err
			}

			fmt.Println(token)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR %s\n", err)
		os.Exit(1)
	}
}

// Generate a signed JWT token that can be used to authenticate as a GitHub App.
// See https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-json-web-token-jwt-for-a-github-app
func generateAppJWT(appID string, pemKey []byte, expiry time.Duration) (string, error) {
	iss := time.Now().Add(-30 * time.Second).Truncate(time.Second)
	exp := iss.Add(expiry).Truncate(time.Second)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": jwt.NewNumericDate(iss),
		"exp": jwt.NewNumericDate(exp),
		"iss": appID,
	})

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemKey)
	if err != nil {
		return "", err
	}

	return token.SignedString(privateKey)
}

func getInstallationID(jwt, repo string) (string, error) {
	resp, err := httpClt.R().
		SetHeader("Accept", "application/json").
		SetAuthToken(jwt).
		Get(fmt.Sprintf("https://api.github.com/repos/%s/installation", repo))
	if err != nil {
		return "", err
	}

	var respData map[string]any
	if err := json.Unmarshal([]byte(resp.String()), &respData); err != nil {
		return "", err
	}
	installationIDRaw, ok := respData["id"]
	if !ok {
		return "", errors.New("installation ID is missing")
	}
	installationID, ok := installationIDRaw.(float64)
	if !ok {
		return "", fmt.Errorf("installation ID %s is not a number", installationIDRaw)
	}
	return strconv.FormatInt(int64(installationID), 10), nil
}

func getAccessToken(jwt, instID string) (string, error) {
	resp, err := httpClt.R().
		SetHeader("Accept", "application/json").
		SetAuthToken(jwt).
		Post(fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", instID))
	if err != nil {
		return "", err
	}

	var respData map[string]any
	if err := json.Unmarshal([]byte(resp.String()), &respData); err != nil {
		return "", err
	}
	tokenRaw, ok := respData["token"]
	if !ok {
		return "", errors.New("access token is missing")
	}
	token, ok := tokenRaw.(string)
	if !ok {
		return "", fmt.Errorf("token %v is not a string", tokenRaw)
	}

	return token, nil
}
