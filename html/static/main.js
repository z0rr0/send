function Copy(elementId) {
    const range = document.createRange();
    range.selectNode(document.getElementById(elementId));
    window.getSelection().removeAllRanges();
    window.getSelection().addRange(range);
    document.execCommand("copy");
    window.getSelection().removeAllRanges();
}

function HumanSize(bytes) {
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    if (bytes === 0) {
        return '0 Byte';
    }
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round(bytes / Math.pow(1024, i)) + ' ' + sizes[i];
}

function successText(data) {
    let content = "<h4>Text data</h4><pre>" + data.text + "</pre>";
    if (data.file !== null) {
        const method = "LoadFile('" + data.file.name + "', '" + data.file.content_type + "')";
        content += "<h4>Download file</h4><a href=\"#\" onclick=\"" + method + "\">" + data.file.name + "</a>&nbsp;";
        content += HumanSize(data.file.size);
    }
    return content;
}

function LoadText(form) {
    const xhttp = new XMLHttpRequest();
    const formData = new FormData();

    formData.append("key", form.key.value);
    formData.append("password", form.password.value);
    xhttp.onreadystatechange = function () {
        if (this.readyState !== 4) {
            return;
        }
        const data = JSON.parse(this.responseText);
        const t = document.getElementById("text_container_id");
        if (this.status === 200) {
            t.innerHTML = "<div class='alert alert-success'>" + successText(data) + "</div>";
        } else {
            t.innerHTML = "<div class='alert alert-danger'><pre>" + data.error + "</pre></div>";
        }
    };
    xhttp.open(form.method, form.action, true);
    xhttp.send(formData);
    return false;
}

function LoadFile(fileName, contentType) {
    const form = document.getElementById("text_form");
    const xhttp = new XMLHttpRequest();
    const formData = new FormData();

    formData.append("key", form.key.value);
    formData.append("password", form.password.value);
    xhttp.onreadystatechange = function () {
        if (this.readyState !== 4) {
            return;
        }
        if (this.status === 200) {
            const blob = new Blob([this.response], {type: contentType});
            const link = document.createElement('a');
            link.href = window.URL.createObjectURL(blob);
            link.download = fileName;
            link.click();
        }
    };
    xhttp.open(form.method, "/file", false);
    xhttp.send(formData);
    return false;
}
