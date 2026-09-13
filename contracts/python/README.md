# aa-contracts Python bindings

Standard-library Python bindings for the v1 schemas, for `aa-sifter` consumers.

- Package: `aa_contracts` (`v1.py`).
- Objects are plain dictionaries validated by `decode`/`validate`; `TypedDict` definitions provide static shape.
- Helpers: `decode`, `encode`, `loads`, `dumps`.

## Test

```sh
cd contracts/python
python -m compileall aa_contracts
python -m unittest discover -s tests -t .
```

The tests round-trip the shared fixtures in `../go/v1/testdata/` and prove invalid objects are rejected.
