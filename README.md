# iapodcast

in case you have a long watch list on Youtube

that mostly consist from podcasts

and these podcasts are long yet supply a lot of infomation

you'd like to capture and even note

----

i want to introduce you to the nice assistant

who can watch them for you

and take notes on the most important topics discussed

providing you with the knowledge from these podcasts

----

use it wisely

it's still learning

keep important for yourself

# build

project depends on [whishper.cpp](https://github.com/ggerganov/whisper.cpp)

fantastic lib that makes transcription extremely fast

to build the project, run

```sh
git submodule update
```

then follow instructions from the linked repo to

- get base (multi-lang model)
- build whishper-cli

then

```sh
ln whishper.cpp/build/bin/whishper-cli whishper-cli
go build .
./iapodcast --url <your-url>
```

