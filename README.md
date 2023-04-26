# Reverse proxy based on a header's value 

(This code is based on the code (and takes inspiration) from: https://github.com/vidosits/header-pattern-proxy)
This middleware can be used to reverse proxy a request based on a headers value, e.g. based on the value of `X-Forwarded-User` from something like [thomseddon/traefik-forward-auth](https://github.com/thomseddon/traefik-forward-auth).

Compared to the original setup, this version allows more complex, dynamic target URLs, partially based on the original URL and the header field. The code also contains better caching, for better performance.


## Configuration

### Static: 
Production:

```toml
[pilot]
  token = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

[experimental]
  [experimental.plugins]
    [experimental.plugins.my-plugin-name]
      moduleName = "github.com/dobots/multiplexer-proxy
      version = "v1.0.0"
```

or if you're using [devMode](https://doc.traefik.io/traefik-pilot/plugins/plugin-dev/#developer-mode):

```toml
[pilot]
  token = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  
[experimental.devPlugin]
  goPath = "/plugins/go"
  moduleName = "github.com/dobots/multiplexer-proxy"
  # Plugin will be loaded from '/plugins/go/src/github.com/dobots/multiplexer-proxy'
```

### Dynamic:

Production:

```toml
[http]
  [http.middlewares]
    [http.middlewares.my-middleware-name.plugin.my-plugin-name]
      header  = "X-Forwarded-User"
      target_match = "^([^.]+).(.*)$"         #If not matching, keep original URL
      target_replace = "$1-${header}.$2"      #Normal (go-lang) regexp rules, ${header} is replaced by the matched header value (URLEncoded)

  [http.routers]
    [http.routers.my-router-name]
      entryPoints = ["websecure"]
      rule = "Host(`my-service-name.domain.tld`)"
      middlewares = ["traefik-forward-auth@docker", "my-middleware-name@file"]
      
      # if no matches are found this is the service that we forward the request to
      service = "noop@internal"
      
      [http.routers.my-router-name.tls]
        certResolver = "letsencrypt"
```

or if you're using [devMode](https://doc.traefik.io/traefik-pilot/plugins/plugin-dev/#developer-mode):

```toml
[http]
  [http.middlewares]
    [http.middlewares.my-middleware-name.plugin.dev]
      header  = "X-Forwarded-User"
      target_match = "^([^.]+).(.*)$"         #If not matching, keep original URL
      target_replace = "$1-${header}.$2"      #Normal (go-lang) regexp rules, ${header} is replaced by the matched header value (URLEncoded)

  [http.routers]
    [http.routers.my-router-name]
      entryPoints = ["websecure"]
      rule = "Host(`my-service-name.domain.tld`)"
      middlewares = ["traefik-forward-auth@docker", "my-middleware-name@file"]
      
      # if no matches are found this is the service that we forward the request to
      service = "noop@internal"
      
      [http.routers.my-router-name.tls]
        certResolver = "letsencrypt"
```

## License
This software is released under the Apache 2.0 License.
