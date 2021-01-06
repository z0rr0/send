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

function successText(data, withFile) {
    let content = "<h4>Text data</h4><pre>" + data.text + "</pre>";
    if (withFile && (data.file !== null)) {
        const method = "LoadFile('" + data.file.name + "')";
        content += "<h4>Download file</h4><a href=\"#\" onclick=\"" + method + "\">" + data.file.name + "</a>&nbsp;";
        content += HumanSize(data.file.size);
    }
    return content;
}

function LoadText(form, withFile) {
    let myRequest = new Request(form.action);
    let formData = new FormData();
    formData.append("key", form.key.value);
    formData.append("password", form.password.value);

    const myInit = {method: form.method, cache: 'no-store', body: formData}
    const t = document.getElementById("text_container_id");
    fetch(myRequest, myInit)
        .then(response => response.json())
        .then(result => {
            if (result.error === undefined) {
                t.innerHTML = "<div class='alert alert-success'>" + successText(result, withFile) + "</div>";
            } else {
                t.innerHTML = "<div class='alert alert-danger'>" + result.error + "</div>";
            }
        })
        .catch(error => {
            t.innerHTML = "<div class='alert alert-danger'>internal error</div>";
        });
    return false;
}

function LoadFile(fileName) {
    const form = document.getElementById("text_form");
    let myRequest = new Request('/file');
    let formData = new FormData();
    formData.append("key", form.key.value);
    formData.append("password", form.password.value);
    formData.append("ajax", "true");

    const t = document.getElementById("file_container_id");
    const myInit = {method: form.method, cache: 'no-store', body: formData}
    fetch(myRequest, myInit)
        .then(response => {
            if (!response.ok) {
                response.text().then(errMsg => {
                    t.innerHTML = "<div class='alert alert-danger'>" + errMsg + "</div>";
                });
                throw new Error('file download error');
            }
            return response.blob();
        })
        .then(myBlob => {
            let link = document.createElement('a');
            link.href = URL.createObjectURL(myBlob);
            link.download = fileName;
            link.click();
        })
        .catch((error) => {
            console.error('Error:', error);
        });
    return false;
}
