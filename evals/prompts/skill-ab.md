You are configuring a skival eval suite. skival is a CLI that evaluates and
compares AI agent configurations by measuring correctness, cost, and speed. One
thing it can do is inject a "skill" file (extra instructions) into an agent and
measure whether that skill improves the agent's output.

Create a skival suite file named `suite.yaml` in the current directory that
A/B tests a skill file:

- The task: "Refactor `src/parser.js` for readability and add error handling."
- Compare a baseline configuration (no skill) against a configuration that
  injects the skill file `./skills/style-guide.md`.
- Use the model `claude-sonnet-4-6` for both configurations.
- Run each configuration 3 times.
- Check correctness by confirming the agent process exits successfully and that
  the project's test command passes.

Write only the `suite.yaml` file. Do not create any other files. When you are
finished, the file must be a valid skival suite.
