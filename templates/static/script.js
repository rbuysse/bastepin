document.addEventListener("DOMContentLoaded", () => {
  const dropArea = document.getElementById("drop-area");
  const centerText = document.getElementById("center-text");
  const spinner = document.getElementById("spinner");

  function ready() {
    centerText.style.display = "block";
    spinner.style.display = "none";
  }

  function busy() {
    centerText.style.display = "none";
    spinner.style.display = "block";
  }

  document.addEventListener("paste", (event) => {
    const clipboardData = event.clipboardData || window.clipboardData;
    const text = clipboardData.getData("text");
    
    if (text) {
      handleTextUpload(text);
    }
  });

  function handleError(errorText) {
    const error = document.getElementById("error");
    error.textContent = `⚠ ${errorText}`;
    error.style.visibility = "visible";

    console.error("something goofed:", errorText);
  }

  async function handleResponse(response) {
    if (response.ok) {
      const url = await response.text();
      window.location.href = url;
    } else {
      const errorText = await response.text();
      handleError(errorText.toLowerCase());
    }
  }

  async function fetchRequest(path, options) {
    busy();
    try {
      const response = await fetch(path, options);
      handleResponse(response);
    } catch (error) {
      console.error("Error:", error);
    }
    ready();
  }

  function handleTextUpload(text) {
    fetchRequest("/upload", {
      method: "POST",
      headers: {
        "Content-Type": "text/plain; charset=utf-8",
      },
      body: text,
    });
  }
});
