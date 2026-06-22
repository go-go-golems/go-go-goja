# Personal Knowledge Inbox tutorial

This example is built incrementally alongside the tutorial in `XGOJA-PERSONAL-INBOX-TUTORIAL`.

Chapter 1 starts with the smallest useful generated xgoja shape: one `xgoja.yaml`, one JavaScript verb, and one generated CLI binary.

Run:

```bash
make doctor
make smoke
```

Manual command after `make build`:

```bash
./dist/personal-knowledge-inbox verbs inbox hello --name tutorial
```
