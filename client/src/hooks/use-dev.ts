function useDev() {
  return window.location.hostname.includes("localhost");
}

export { useDev };
