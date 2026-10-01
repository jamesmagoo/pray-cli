# Pray CLI
Pray cli allows you to bring prayer into the command line interface - whether that be for a peaceful moment of contemplation with The Lord, or for easily inserting prayers into your work (a logged prayer to St. Carlo Acutis never hurt a shell script or server's reliability).

```
╔════════════════════════════════════════════════════════════════════════════════╗
║                                                                                ║
║                                      ███                                       ║
║                                      ███                                       ║
║                                ███████████████                                 ║
║                                      ███                                       ║
║                                      ███                                       ║
║                                      ███                                       ║
║                                      ███                                       ║
║                                                                                ║
║                  Prayer of Intercession for Technical Problems                 ║
║                                                                                ║
║    Almighty and eternal God, you created us in your image, and you bid us      ║
║    seek all that is good and true and beautiful, fully manifest only in the    ║
║    divine person of your only-begotten Son, Jesus Christ.                      ║
║                                                                                ║
║    You have inspired your servant, St. Carlo Acutis, to seek Your goodness,    ║
║    to proclaim Your truth, and to portray Your beauty through technology.      ║
║                                                                                ║
║    I humbly ask your servant's prayers, that I too may lead others to you      ║
║    through technology.                                                         ║
║                                                                                ║
║    I humbly ask, through the intercession of St. Carlo, that you grant me      ║
║    clarity and patience in my technological work, specifically for "pray cli". ║
║                                                                                ║
║    Enlighten my understanding and direct my hands in every design and in       ║
║    every line of code, that my work may always serve your greater glory        ║
║    and benefit those who will use what I create.                               ║
║                                                                                ║
║                         Through Christ our Lord. Amen.                         ║
║                                                                                ║
╚════════════════════════════════════════════════════════════════════════════════╝
```

## Getting started

### Install

You'll need Go 1.25 or newer.

```sh
go install github.com/jamesmagoo/pray-cli/cmd/pray@latest
```

This puts `pray` in `$(go env GOPATH)/bin`, so make sure that folder is on your `PATH`.

Or build it from a clone:

```sh
git clone https://github.com/jamesmagoo/pray-cli
cd pray-cli
just install        # or: go install ./cmd/pray
```

### See what's available

```sh
pray list
```

```
hail-mary         Hail Mary                                       en la
our-father        Our Father                                      en la
st-carlo-acutis   Prayer of Intercession for Technical Problems   en
```

Each line shows the prayer's name, its title and the languages it's
available in. Running `pray` on its own shows the same list. Narrow it down
by tag, or see the titles in Latin:

```sh
pray list --tag marian
pray list --lang la
```

### Pray

Type `pray` and the name of a prayer:

```sh
pray hail mary
```

```
Hail Mary

Hail Mary, full of grace,
the Lord is with thee.
Blessed art thou amongst women,
and blessed is the fruit of thy womb, Jesus.

Holy Mary, Mother of God,
pray for us sinners,
now and at the hour of our death.

Amen.
```

You don't need the exact name. `pray` matches a prayer's name, title and nicknames, and the start of any word in them:

```sh
pray ave            # a nickname for the Hail Mary
pray carlo          # the Prayer of Intercession for Technical Problems
pray saint carlo    # "saint", "st" and "st." all work
pray acut           # the start of a word is enough
```

If what you type could mean more than one prayer, `pray` lists the options so you can be more specific.

### Pray for someone or something

Add an intention with `--for`:

```sh
pray hail mary --for "my mum"
pray carlo --for "the server migration"
```

Some prayers have a place for the intention in their text. The St. Carlo prayer, for example, asks for help "specifically for the server migration". Otherwise the intention is named before the prayer, as you would say it aloud:

```
Hail Mary

For my mum

Hail Mary, full of grace,
…
```

### Pray in Latin

```sh
pray hail mary --lang la      # Ave Maria
```

If a prayer hasn't been translated yet, you get the English.

### Copy a prayer

```sh
pray copy hail mary
pray copy carlo --for "the release"
```

This puts the prayer on your clipboard as plain text, ready to paste into a message, a document or a commit. On Linux it needs `xclip`, `xsel` or `wl-clipboard` installed.

### Use it in scripts

When the output is piped or redirected, `pray` always writes plain text, so it's safe to log:

```sh
#!/bin/sh
pray carlo --for "tonight's deploy" >> deploy.log
./deploy.sh
```

### Options

| Option | What it does |
|---|---|
| `--for "<intention>"` | Pray for someone or something |
| `--lang <code>` | Language: `en` (default) or `la` |
| `--tag <tag>` | With `pray list`: only prayers with this tag, e.g. `marian` |
| `-h`, `--help` | Help for `pray` or any command, e.g. `pray copy --help` |

# Coming Soon...
1. `rosary` mode! 
2. more prayers
3. more languages
4. your contributions!
