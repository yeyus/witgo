# witgo
WIP. Go client for wit.ai

## Info

- [v1 Docs](https://godoc.org/github.com/kurrik/witgo/v1/witgo)

## Usage

Import the versioned path:

    import "github.com/kurrik/witgo/v1/witgo"

Should be close to the
[Node wit.ai client library](https://github.com/wit-ai/node-wit).

Construct a handler satisfying the following interface:

    type Handler interface {
            Action(session *Session, action string) (response *Session, err error)
            Say(session *Session, msg string) (response *Session, err error)
            Merge(session *Session, entities EntityMap) (response *Session, err error)
            Error(session *Session, msg string)
    }

Create a client with your Server Access Token:

    client = witgo.NewClient(token)

Create an input reader satisfying the following interface:

    type Input interface {
            Run() (requests chan<- SessionID, records <-chan InputRecord)
    }

Or use the interactive input reader:

    input = witgo.NewInteractiveInput()

Then create a new `Witgo` object and call process on it:

    wg = witgo.NewWitgo(client, handler)
    err = wg.Process(input)


## Environment flags

| Flag | Description |
| ---- | ----------- |
| HTTP_PROXY | Passes API requests through a proxy, useful for debugging.  Ex: `HTTP_PROXY=http://localhost:8080` |
| TLS_INSECURE | **Do not use in production!** Ignore SSL errors.  May help if you're using a proxy and requests are rejected due to certificate errors. Ex: `TLS_INSECURE=1` |

## Examples

The `examples` directory contain directories which demonstrate some of the intended uses of this library:

 * [Interactive example](./examples/01-weather/README.md)
 * [Twitter example](./examples/02-twitter/README.md)
