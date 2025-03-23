import { ensureDirSync } from "https://deno.land/std/fs/mod.ts";

const Config = {
  systemPrompts: {
    ideate: Deno.readTextFileSync("./prompts/ideate.txt").trim(),
    verify: Deno.readTextFileSync("./prompts/verify.txt").trim(),
  },
  models: {
    claude: "Claude 3.7 Sonnet",
  },
  questionsDir: "./questions/",
};

class QuestionGenerator {
  state = "initial"; // one of: initial, ideate, verify, done
  question = null;
  verificationResult = null;
  previousQuestions = [];

  constructor() {
    // Ensure questions directory exists
    ensureDirSync(Config.questionsDir);
    this.loadPreviousQuestions();
  }

  loadPreviousQuestions() {
    try {
      // Read all files from the questions directory
      const files = [...Deno.readDirSync(Config.questionsDir)].filter(
        (file) => file.isFile && file.name.endsWith(".json")
      );

      console.log(`Found ${files.length} previous questions`);

      // Load and parse each file
      for (const file of files) {
        try {
          const content = Deno.readTextFileSync(`${Config.questionsDir}${file.name}`);
          const question = JSON.parse(content);
          this.previousQuestions.push(question);
        } catch (error) {
          console.warn(`Failed to parse question file ${file.name}:`, error);
        }
      }
    } catch (error) {
      console.error("Error loading previous questions:", error);
    }
  }

  formatPreviousQuestionsForPrompt() {
    if (this.previousQuestions.length === 0) {
      return "No previous questions generated yet.";
    }

    return this.previousQuestions
      .map((q, i) => {
        return `
<previous_question id="${i + 1}">
Title: ${q.title}
Problem: ${q.problem}
Options:
${q.options.join("\n")}
Correct Answer: ${q.answer.correct}
</previous_question>
      `.trim();
      })
      .join("\n\n");
  }

  async run() {
    try {
      while (this.state !== "done") {
        console.log(`\n=== Current state: ${this.state} ===`);
        switch (this.state) {
          case "initial":
            this.state = "ideate";
            break;
          case "ideate":
            await this.onIdeate();
            break;
          case "verify":
            await this.onVerify();
            break;
        }
      }

      // Save the final verified question to a file
      if (
        this.question &&
        this.verificationResult &&
        this.verificationResult.verdict === "CORRECT"
      ) {
        this.saveQuestionToFile();
      }

      return {
        question: this.question,
        verification: this.verificationResult,
      };
    } catch (error) {
      console.error("Error in question generation process:", error);
      return {
        error: error.message,
        state: this.state,
        question: this.question,
        verification: this.verificationResult,
      };
    }
  }

  saveQuestionToFile() {
    const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
    const filename = `${Config.questionsDir}${timestamp}.json`;

    try {
      Deno.writeTextFileSync(filename, JSON.stringify(this.question, null, 2));
      console.log(`Question saved to ${filename}`);
    } catch (error) {
      console.error("Failed to save question to file:", error);
    }
  }

  async onIdeate() {
    console.log("Starting ideation phase...");

    // Create a modified system prompt that includes previous questions
    const previousQuestionsSection =
      this.previousQuestions.length > 0
        ? `
## Previously Generated Questions
The following questions have already been generated. Create something DIFFERENT and AVOID repeating similar concepts:

${this.formatPreviousQuestionsForPrompt()}
      `.trim()
        : "";

    const enhancedSystemPrompt = `${Config.systemPrompts.ideate}

${previousQuestionsSection}

## Diversity Requirements
- Your question MUST be substantially different from all previous questions
- Explore different mathematical concepts, difficulty levels, or problem structures
- If previous questions focused on certain topics, try to cover new areas
- Ensure your question is unique in its approach and solution method
`;

    const ideationPrompt = `
Generate 3 different practice questions for AMK valintakoe (Finnish University of Applied Sciences entrance exam).
Each question should be challenging but fair.
Include questions from different subject areas if possible.
IMPORTANT: Make sure your questions are different from the ${this.previousQuestions.length} previously generated questions.
Do not over-index on "lukujono" style questions.
    `.trim();

    console.log("Requesting initial ideas from model...");
    const resp1 = await this.callModel(Config.models.claude, enhancedSystemPrompt, [
      { role: "user", content: ideationPrompt },
    ]);

    console.log("\n--- Generated Ideas ---");
    console.log(resp1);
    console.log("--- End of Ideas ---\n");

    console.log("Requesting best idea selection in JSON format...");
    const selectionPrompt = `
Based on the ideas you just generated, select the best question candidate and format it as JSON with the following structure:

{
  "title": "Question title",
  "problem": "Full problem statement",
  "options": [
    "a) Option A",
    "b) Option B",
    "c) Option C",
    "..."
  ],
  "answer": {
    "explanation": "Explanation of the correct answer, written in a way that a student can understand and learn from it.",
    "correct": "b",
  }
}

Choose the question that is most representative of the actual exam, has clear instructions, can be solved without a calculator, and tests important skills.

Your answer MUST begin with '{' and end with '}'.
The selected question MUST be written in Finnish.
The question title, problem statement, and explanation should be in Finnish, and SHOULD NOT include ANY formatting whatsoever (no markdown, no tables, etc.).
Ensure it is DIFFERENT from all previously shown questions.
    `.trim();

    const jsonResponse = await this.callModel(Config.models.claude, enhancedSystemPrompt, [
      { role: "user", content: ideationPrompt },
      { role: "assistant", content: resp1 },
      { role: "user", content: selectionPrompt },
    ]);

    console.log("\n--- JSON Response ---");
    console.log(jsonResponse);
    console.log("--- End of JSON Response ---\n");

    try {
      // Extract JSON if it's wrapped in markdown or other text
      const jsonMatch =
        jsonResponse.match(/```(?:json)?\s*([\s\S]*?)\s*```/) || jsonResponse.match(/({[\s\S]*})/);

      const jsonString = jsonMatch ? jsonMatch[1] || jsonMatch[0] : jsonResponse;
      const parsedJson = JSON.parse(jsonString.trim());

      console.log("Successfully parsed JSON response");
      this.question = parsedJson;
      this.state = "verify";
    } catch (error) {
      console.error("Failed to parse JSON response:", error);
      console.log("Attempting fallback parsing...");

      // Fallback: Try to extract anything that looks like JSON
      try {
        const fallbackMatch = jsonResponse.match(/{[\s\S]*}/);
        if (fallbackMatch) {
          const fallbackJson = JSON.parse(fallbackMatch[0]);
          console.log("Fallback parsing successful");
          this.question = fallbackJson;
          this.state = "verify";
        } else {
          throw new Error("No valid JSON found in response");
        }
      } catch (fallbackError) {
        console.error("Fallback parsing failed:", fallbackError);
        throw new Error(`JSON parsing failed: ${error.message}`);
      }
    }

    console.log("\n--- Selected Question ---");
    console.log(JSON.stringify(this.question, null, 2));
    console.log("--- End of Selected Question ---\n");
  }

  async onVerify() {
    console.log("Starting verification phase...");

    const verifyPrompt = `
Verify the following AMK valintakoe practice question:

TITLE: ${this.question.title}

PROBLEM: ${this.question.problem}

OPTIONS:
${this.question.options.join("\n")}

CLAIMED CORRECT ANSWER: ${this.question.answer.correct}

EXPLANATION: ${this.question.answer.explanation}

Analyze this question thoroughly. Check for:
1. Mathematical correctness
2. Clarity of the problem statement
3. Whether the claimed correct answer is actually correct
4. Whether the explanation is clear and accurate
5. Appropriate difficulty level for AMK entrance exam
6. Solvability without a calculator

Provide your analysis, and end with one of these verdicts:
- VERDICT: CORRECT - if the question is valid and the answer is correct
- VERDICT: INCORRECT - if there are any issues with the question or answer
- VERDICT: NEEDS REVISION - if the question is mostly good but needs specific changes

If you choose INCORRECT or NEEDS REVISION, explain exactly what needs to be fixed.
    `.trim();

    console.log("Requesting verification from model...");
    const verificationResponse = await this.callModel(
      Config.models.claude,
      Config.systemPrompts.verify,
      [{ role: "user", content: verifyPrompt }]
    );

    console.log("\n--- Verification Response ---");
    console.log(verificationResponse);
    console.log("--- End of Verification Response ---\n");

    // Extract the verdict
    const verdictMatch = verificationResponse.match(
      /VERDICT:\s*(CORRECT|INCORRECT|NEEDS REVISION)/i
    );
    const verdict = verdictMatch ? verdictMatch[1] : "UNKNOWN";

    this.verificationResult = {
      verdict: verdict,
      analysis: verificationResponse,
    };

    // If needed, we could implement automatic revision here for NEEDS REVISION cases

    this.state = "done";
  }

  async callModel(model, system, messages) {
    try {
      const resp = await fetch("http://localhost:8765/api", {
        method: "POST",
        headers: { Authorization: "Bearer YOUR_TOKEN_HERE" },
        body: JSON.stringify({ model, system, messages }),
      });

      if (!resp.ok) {
        const errorText = await resp.text();
        throw new Error(`API error (${resp.status}): ${errorText}`);
      }

      const data = await resp.json();
      return data.message;
    } catch (error) {
      console.error(`Error calling model ${model}:`, error);
      throw new Error(`Model call failed: ${error.message}`);
    }
  }
}

// Execute and log the result
const generator = new QuestionGenerator();
generator
  .run()
  .then((result) => {
    console.log("\n=== FINAL RESULT ===");
    console.log(`Question: ${result.question?.title || "None generated"}`);
    console.log(`Verification: ${result.verification?.verdict || "Not verified"}`);
    console.log("\nComplete output:");
    console.log(JSON.stringify(result, null, 2));
  })
  .catch((error) => {
    console.error("Fatal error:", error);
  });
