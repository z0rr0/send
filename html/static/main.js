function Copy(elementId) {
    const range = document.createRange();
    range.selectNode(document.getElementById(elementId));
    window.getSelection().removeAllRanges();
    window.getSelection().addRange(range);
    document.execCommand("copy");
    window.getSelection().removeAllRanges();
}
