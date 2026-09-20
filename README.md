_Читать на русском: [README.ru.md](README.ru.md)_

# clank

A small CLI agent for the terminal.

```bash
dmesg | tail -50 | clank "what's wrong here"
clank "why doesn't grub see the second disk"
```

Works with any OpenAI-compatible API: cloud providers, local Ollama / LocalAI / vLLM.


## What it can do

- Answers questions in plain text, no markdown garbage (well, it tries).
- Reads context from a pipe: logs, diffs, output of any commands.
- Remembers the conversation within the terminal tab — `clank -r "now this way"`.
- Can run commands itself to figure things out.
- If the question is unclear — asks back, suggests options.
- Understands images: `clank -i screen.png "what's wrong here"`, or asks permission to read them (e.g. for `clank "find all pictures with triangles"` it loads them itself).


## Installation

Ready-made builds are on the releases page: https://github.com/Gangsta0607/clank/releases


## Install from source

```bash
git clone https://github.com/Gangsta0607/clank.git
cd clank
make
sudo make install
```


## Quick start

```bash
clank config init     # asks for the API address, key and model, saves a profile
clank "connection check"
```

If you have several providers — create a profile for each (`clank config add ...`) and switch between them (`clank config use ...`).


The full list of commands and flags is in the built-in help:

```bash
clank help
```

## License

MIT / Public Domain.
