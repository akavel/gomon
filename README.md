gomon
=====

[![Build Status](https://travis-ci.org/akavel/gomon.png)](https://travis-ci.org/akavel/gomon)

Gomon is a files and directories monitor, which starts specified command automatically
after any change is detected.

Install
-------

    go get -u github.com/akavel/gomon

Usage
-----

    Usage: gomon [OPTIONS] [DIR] -- COMMAND [WITH ARGS...]
    Where OPTIONS:
      -include="\\.(go|c|h)$": regular expressions specifying file patterns to watch


Example - Monitoring With Custom Command:

    gomon -- go build # monitor current directory recursively, build if changed


Contributors
------------

- Ask Bjørn Hansen
- Yasuhiro Matsumoto (a.k.a mattn)
- Mateusz Czapliński

License
--------

MIT License

