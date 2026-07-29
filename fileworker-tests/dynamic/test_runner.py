import importlib.util
from pathlib import Path
import unittest

MODULE_PATH = Path(__file__).with_name("runner.py")
SPEC = importlib.util.spec_from_file_location("dynamic_runner", MODULE_PATH)
runner = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(runner)


class RunnerUnitTest(unittest.TestCase):
    def test_loopback_urls(self):
        self.assertEqual(runner.loopback_url("http://127.0.0.1:8099/"),
                         "http://127.0.0.1:8099")
        self.assertEqual(runner.loopback_url("http://localhost:8765"),
                         "http://localhost:8765")

    def test_non_loopback_rejected(self):
        with self.assertRaises(Exception):
            runner.loopback_url("https://example.com")

    def test_extract_structured_output(self):
        value = {"verdict": "PASS"}
        self.assertEqual(runner.extract_result({"structured_output": value}), value)
        self.assertEqual(runner.extract_result({"result": '{"verdict":"FAIL"}'}),
                         {"verdict": "FAIL"})
        self.assertIsNone(runner.extract_result({"result": "not json"}))


if __name__ == "__main__":
    unittest.main()
