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


## Running examples

Obtain your `Server Access Token` from your wit.ai project settings:

    export SERVER_ACCESS_TOKEN=xxxx

Run the weather example:

    $ ./scripts/run.sh 01-weather -token=$SERVER_ACCESS_TOKEN
    Running example 'examples/01-weather' with args '-token=...'
    Interactive mode (use ':quit' to stop)
    interactive> What is the weather?
    < Where, exactly?
    interactive> In Paris?
    < I see it’s sunny in Paris today!
    interactive> :quit

This example mimics the functionality covered in the
[wit.ai quick start tutorial](https://wit.ai/docs/quickstart).  You can view
the source of the example [here](/examples/weather/main.go).

For debug output including raw HTTP logs:

    ./scripts/run.sh 01-weather -debug -token=$SERVER_ACCESS_TOKEN
