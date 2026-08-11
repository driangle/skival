You are configuring a skival eval suite. skival is a CLI that evaluates and
compares AI agent configurations by measuring correctness, cost, and speed.

Create a skival suite file named `suite.yaml` in the current directory that
compares two Claude models on a single coding task:

- The task: "Write a Python function `is_palindrome(s)` that returns whether a
  string is a palindrome, ignoring case and non-alphanumeric characters."
- Compare the models `claude-sonnet-4-6` and `claude-opus-4-6` head-to-head.
- Run each configuration 3 times so the results are statistically meaningful.
- Check correctness by confirming the agent process exits successfully and that
  the produced code passes a test command.

Write only the `suite.yaml` file. Do not create any other files. When you are
finished, the file must be a valid skival suite.
