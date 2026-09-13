"""Round-trip and rejection tests for the Python contract bindings."""

import json
import pathlib
import unittest

from aa_contracts import decode, dumps, loads

TESTDATA = pathlib.Path(__file__).resolve().parents[2] / "go" / "v1" / "testdata"


class RoundTripTest(unittest.TestCase):
    def test_valid_fixtures_round_trip(self):
        files = sorted(TESTDATA.glob("*.json"))
        self.assertTrue(files, "no fixtures found")
        for path in files:
            if "invalid" in path.name:
                continue
            with self.subTest(path.name):
                data = json.loads(path.read_text(encoding="utf-8"))
                obj = decode(data)
                again = loads(dumps(obj))
                self.assertEqual(obj, again)

    def test_invalid_fixtures_are_rejected(self):
        files = sorted(TESTDATA.glob("*invalid*.json"))
        self.assertTrue(files, "expected at least one invalid fixture")
        for path in files:
            with self.subTest(path.name):
                with self.assertRaises(ValueError):
                    decode(json.loads(path.read_text(encoding="utf-8")))

    def test_unknown_kind_is_rejected(self):
        with self.assertRaises(ValueError):
            decode({"contractVersion": "1.0", "kind": "nope", "id": "x", "title": "x"})

    def test_missing_contract_version_is_rejected(self):
        with self.assertRaises(ValueError):
            decode({"kind": "issue", "id": "iss_1", "title": "x", "type": "bug", "status": "open"})


if __name__ == "__main__":
    unittest.main()
