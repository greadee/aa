import tempfile
import unittest
from pathlib import Path

from check_binding_parity import diff_models, load_go_dir, parse_py, parse_ts


def go_struct(*, fields=None, embedded=None):
    return {
        "kind": "struct",
        "fields": fields or [],
        "embedded": embedded or [],
    }


def go_field(name, go_type, optional=False):
    return {
        "jsonName": name,
        "goName": name[:1].upper() + name[1:],
        "goType": go_type,
        "optional": optional,
        "pointer": go_type.startswith("*"),
    }


class BindingParityTest(unittest.TestCase):
    def test_matching_models_have_no_problems(self):
        go = {"types": {"T": go_struct(fields=[go_field("id", "string"), go_field("name", "string", True)])}}
        ts = parse_ts("""
export interface T {
  id: string;
  name?: string;
}
""")
        py = parse_py("""
class T(TypedDict, total=True):
    id: str
    name: NotRequired[str]
""")
        self.assertEqual(diff_models(go, ts, py), [])

    def test_name_only_parity_is_not_enough(self):
        go = {"types": {"T": go_struct(fields=[go_field("id", "string")])}}
        ts = parse_ts("export interface T {\n  other: string;\n}\n")
        py = parse_py("""
class T(TypedDict, total=True):
    other: str
""")
        problems = diff_models(go, ts, py)
        self.assertTrue(any("TS T: missing field id" in p for p in problems), problems)
        self.assertTrue(any("PY T: missing field id" in p for p in problems), problems)
        self.assertTrue(any("TS T: undocumented field other" in p for p in problems), problems)
        self.assertTrue(any("PY T: undocumented field other" in p for p in problems), problems)

    def test_type_mismatch_is_detected(self):
        go = {"types": {"T": go_struct(fields=[go_field("id", "string")])}}
        ts = parse_ts("export interface T {\n  id: number;\n}\n")
        py = parse_py("""
class T(TypedDict, total=True):
    id: int
""")
        problems = diff_models(go, ts, py)
        self.assertTrue(any("TS T.id: type mismatch" in p for p in problems), problems)
        self.assertTrue(any("PY T.id: type mismatch" in p for p in problems), problems)

    def test_optionality_mismatch_is_detected(self):
        go = {"types": {"T": go_struct(fields=[go_field("id", "string", False)])}}
        ts = parse_ts("export interface T {\n  id?: string;\n}\n")
        py = parse_py("""
class T(TypedDict, total=False):
    id: str
""")
        problems = diff_models(go, ts, py)
        self.assertTrue(any("TS T.id: optionality mismatch" in p for p in problems), problems)
        self.assertTrue(any("PY T.id: optionality mismatch" in p for p in problems), problems)

    def test_ts_extends_and_python_bases_are_expanded(self):
        go = {"types": {
            "Base": go_struct(fields=[go_field("id", "string")]),
            "Child": go_struct(fields=[go_field("name", "string")], embedded=["Base"]),
        }}
        ts = parse_ts("""
export interface Base {
  id: string;
}
export interface Child extends Base {
  name: string;
}
""")
        py = parse_py("""
class Base(TypedDict, total=True):
    id: str
class Child(Base, total=True):
    name: str
""")
        self.assertEqual(diff_models(go, ts, py), [])

    def test_grouped_go_declarations_are_parsed_with_go_ast(self):
        source = """package v2

type (
    Outer struct {
        Envelope
        Name string `json:"name"`
    }
    Other struct {
        ID string `json:"id,omitempty"`
    }
)

type Envelope struct {
    ID string `json:"id"`
}
"""
        with tempfile.TemporaryDirectory() as directory:
            Path(directory, "grouped.go").write_text(source)
            model = load_go_dir(Path(directory))
        self.assertEqual(model["types"]["Outer"]["kind"], "struct")
        self.assertEqual(model["types"]["Outer"]["embedded"], ["Envelope"])
        self.assertEqual(model["types"]["Other"]["fields"][0]["jsonName"], "id")
        self.assertTrue(model["types"]["Other"]["fields"][0]["optional"])

    def test_ts_alias_union_resolves_to_string(self):
        ts = parse_ts("""
export type State = "ready" | "blocked";
export interface T {
  state: State;
}
""")
        go = {"types": {"T": go_struct(fields=[go_field("state", "string")])}}
        py = parse_py("""
class T(TypedDict, total=True):
    state: str
""")
        self.assertEqual(diff_models(go, ts, py), [])

    def test_python_class_key_alias_maps_to_class(self):
        py = parse_py("""
class Failure(TypedDict, total=False):
    class_: str
""")
        self.assertIn("class", py["classes"]["Failure"]["fields"])
        self.assertNotIn("class_", py["classes"]["Failure"]["fields"])

    def test_arrays_and_maps_compare_by_category(self):
        go = {"types": {
            "Reference": go_struct(fields=[go_field("id", "string")]),
            "T": go_struct(fields=[
                go_field("items", "[]Reference"),
                go_field("metadata", "map[string]any"),
            ]),
        }}
        ts = parse_ts("""
export interface Reference { id: string; }
export interface T {
  items: Reference[];
  metadata: Record<string, unknown>;
}
""")
        py = parse_py("""
class Reference(TypedDict, total=True):
    id: str
class T(TypedDict, total=True):
    items: List[Dict[str, Any]]
    metadata: Dict[str, Any]
""")
        self.assertEqual(diff_models(go, ts, py), [])

    def test_typescript_or_python_only_type_is_detected(self):
        go = {"types": {"T": go_struct(fields=[go_field("id", "string")])}}
        ts = parse_ts("""
export interface T { id: string; }
export interface Extra { id: string; }
""")
        py = parse_py("""
class T(TypedDict, total=True):
    id: str
class Extra(TypedDict, total=True):
    id: str
""")
        problems = diff_models(go, ts, py)
        self.assertTrue(any("TypeScript declaration without Go struct: Extra" in p for p in problems), problems)
        self.assertTrue(any("Python TypedDict without Go struct: Extra" in p for p in problems), problems)

    def test_required_wrapper_wins_inside_total_false(self):
        go = {"types": {"T": go_struct(fields=[go_field("id", "string", False)])}}
        ts = parse_ts("export interface T {\n  id: string;\n}\n")
        py = parse_py("""
class T(TypedDict, total=False):
    id: Required[str]
""")
        self.assertEqual(diff_models(go, ts, py), [])


if __name__ == "__main__":
    unittest.main()
