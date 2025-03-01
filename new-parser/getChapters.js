// visit https://ec.europa.eu/taxation_customs/dds2/taric/taric_consultation.jsp?Lang=lt&Expand=true&SimDate=20250225#afterForm, select language and prefill chapters
// Script to extract values from .tddescription elements and format as SQL insert statements
function extractAndFormatDescriptions() {
  // Get all elements with class .tddescription
  const descriptions = document.querySelectorAll(
    ".section_heading .tddescription"
  );
  let result = "";

  // Process each element
  descriptions.forEach((element, index) => {
    // Get the text content and clean it up
    const value = element.textContent.trim().replace(/'/g, "''"); // Escape single quotes for SQL

    // Format as SQL insert statement with index+1 (to start from 1 instead of 0)
    result += `(${index + 1}, 'EN', '${value}'),\n`;
  });

  // Remove trailing comma and newline
  result = result.replace(/,\n$/, ";");

  // Output to console for easy copying
  console.log(result);

  // Return the formatted string
  return result;
}

// Execute the function and store the result
const sqlInsertStatements = extractAndFormatDescriptions();

// Create a temporary textarea to allow easy copying of the result
const textarea = document.createElement("textarea");
textarea.value = sqlInsertStatements;
document.body.appendChild(textarea);
textarea.select();
document.execCommand("copy");
document.body.removeChild(textarea);

console.log("SQL insert statements copied to clipboard!");
