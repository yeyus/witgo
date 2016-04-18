# Interactive example

This example demonstrates a very simple integration with wit.ai.  It implements the same functionality as the [wit.ai quick start](https://wit.ai/docs/quickstart).

See [main.go](./main.go) for source code.

## Running

Obtain your `Server Access Token` from your wit.ai project settings:

    export SERVER_ACCESS_TOKEN=xxxx

Run the weather example:

    ./scripts/run.sh 01-weather -token=$SERVER_ACCESS_TOKEN

You will be able to interact with your wit.ai app on the command line:

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
