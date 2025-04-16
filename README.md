# JIT Example

This is an example of an application to automate Just-In-Time access with Pomerium Zero. See the [Just-In-Time Access Guide](https://www.pomerium.com/docs/guides/jit) for more details.

## Setup

This application is intended to be run behind a Pomerium instance using Pomerium Zero. It needs two routes configured in Pomerium Zero: (In the below examples we have used the `curious-cat-9999.pomerium.app` starter domain. You should replace this with your own starter domain or use a custom domain.)

- `jit-example.curious-cat-9999.pomerium.app`
  - **From** should be `https://jit-example.curious-cat-9999.pomerium.app`
  - **To** should point to your instance of jit-example
  - **Any Authenticated User** should be the attached policy
  - In **Headers**, **Pass Identity Headers** needs to be enabled
- `jit-example.curious-cat-9999.pomerium.app/admin`
  - **From** should be `https://jit-example.curious-cat-9999.pomerium.app`
  - **To** should point to your instance of jit-example
  - In **Path Matching**, **Prefix** should be set to `/admin`
  - In **Path Rewriting**, **Prefix Rewrite** should be set to `/admin`
  - A policy restricting the users to administrators only should be attached policy
  - In Headers, **Pass Identity Headers** needs to be enabled

With this setup any user will have access to `https://jit-example.curious-cat-9999.pomerium.app` but only administrators will be able to approve access requests.

## Running

To run the application you will need the following environment variables:

- `API_USER_TOKEN`: The Pomerium Zero API User Token used to access the Pomerium Zero API. You can create one [here](https://console.pomerium.app/app/management/api-tokens).
- `CLUSTER_ID`: The ID of the cluster where the JIT example policies will be created. A list of your clusters is available [here](https://console.pomerium.app/app/clusters).
- `JWKS_ENDPOINT`: The URL of the Pomerium route sitting in front of the JIT example application. For example: `http://jit-example.curious-cat-9999.pomerium.app/.well-known/pomerium/jwks.json`.
- `ORGANIZATION_ID`: The ID of your organization.
- `PORT`: The port to run the web server on.

With those environment variables set you can run the application via:

```
go run .
```

There is also a Dockerfile available that can be used to build a docker image listening on port 8000.

## Routes

The following endpoints are available:

- `/`: the index page where users can request access
- `/admin`: the admin page where administrators can approve access requests

## Policy

When users are granted access a `jit-example` policy is created. Assign this policy to routes and any users granted access will be able to access those routes.
