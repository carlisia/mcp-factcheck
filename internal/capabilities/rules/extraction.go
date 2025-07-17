package rules

// ClaimExtraction defines how to extract claims from content.
// This constant provides detailed rules for parsing and extracting individual
// claims from compound sentences and lists in MCP-related content.
const ClaimExtraction = `- Your output will be considered incorrect if you miss, combine, or summarize any claim or list item, or if you output a claim for only the first & last item in a list but not every item.
- Identify every claim about MCP, even if phrased as a fragment, list item, or implicit subject.
- When encountering a clause with a subject & verb followed by a list (e.g., "enforces voice recognition, database integration, and blockchain validation"), create a separate claim for every item in the list by combining the subject & verb with each item.
    - For example:  
      - Input: "Implements blockchain validation; supports distributed file storage, voice recognition, & NoSQL database integration."
      - Output claims:
        - "MCP implements blockchain validation"
        - "MCP supports distributed file storage"
        - "MCP supports voice recognition"
        - "MCP supports NoSQL database integration"
- For every list (e.g., "supports distributed file storage, voice recognition, & NoSQL database integration"), output a separate claim for each item in the list.
- Do not skip any item—not the first, middle, or last. Skipping a list item is a critical error.
- The output claims must match the count of list items exactly.
- If any item in a list appears in the user content, your output must contain a separate, fully expanded claim for that item, even if the list contains two items, three items, or more.
- If you output claims for the first & last items of a list but not all intermediate items, this is a critical error. Every list item must become its own claim.
- If a claim lacks a subject, assume "MCP" as the subject.
- If a claim lacks a verb but is part of a list or compound sentence, inherit the verb from the prior clause.
    - Example: "supports distributed file storage, voice recognition, & NoSQL database integration" becomes three claims, each beginning with "MCP supports".
- Preserve the original order of claims as found in the text.
- Do not omit any item from lists or compound sentences.
- Treat each bullet or numbered item as a separate claim.
- For each claim, output the fully expanded version with subject & verb included.
- The number of output claims must exactly match the number of items in the input lists, plus any other standalone claims.
- If your output does not contain a separate claim for each item in every user content list, your answer is incorrect, even if the other claims are correct.`