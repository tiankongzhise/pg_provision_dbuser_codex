const passwordButton = document.querySelector("#generate-password");
const passwordInput = document.querySelector("#role-password");

if (passwordButton && passwordInput) {
  passwordButton.addEventListener("click", async () => {
    passwordButton.disabled = true;
    try {
      const response = await fetch("/api/password", {
        headers: { Accept: "application/json" },
      });
      if (!response.ok) {
        throw new Error("request failed");
      }
      const data = await response.json();
      passwordInput.value = data.password || "";
      passwordInput.type = "text";
      passwordInput.focus();
      passwordInput.select();
    } finally {
      passwordButton.disabled = false;
    }
  });
}
