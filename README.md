# github-app-token

`github-app-token` is a simple CLI tool that generates an app token so that you can authenticate as a GitHub App.


## Usage

1. Set the necessary secrets as environment variables:
     - `GITHUB_APP_ID`
     - `GITHUB_APP_PRIVATE_KEY`

1. Run `github-app-token` and the generated token will be emitted to stdout.


## Alternatives

- If using in GitHub Actions, use [tibdex/github-app-token](https://github.com/tibdex/github-app-token)
