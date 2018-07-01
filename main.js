const errorElement = document.getElementById("error");
function logImportError(message) {
	errorElement.innerText = "Error while importing: " + message;
}

function handleString(input) {
	console.log("Found " + input.length);

	dropbox.classList.add("d-none");
}

// Read string data from file
function handleFile(file) {
	let reader = new FileReader();

    reader.onload = function(e) {
        let text = reader.result;
        handleString(text);
	};
	
	reader.onerror = function(e) {
		logImportError(reader.error.message);
	}

    // Read the manifest file as text
	reader.readAsText(file);
}

// Drop area
const dropbox = document.getElementById("importArea");
dropbox.addEventListener("dragenter", (e) => {
	e.stopPropagation();
	e.preventDefault();
	dropbox.classList.add("dragging");
}, false);
dropbox.addEventListener("dragleave", (e) => {
	e.stopPropagation();
	e.preventDefault();
	dropbox.classList.remove("dragging");
}, false);
dropbox.addEventListener("dragover", (e) => {
	e.stopPropagation();
	e.preventDefault();
	dropbox.classList.add("dragging");
}, false);
dropbox.addEventListener("drop", (e) => {
	e.stopPropagation();
	e.preventDefault();

	dropbox.classList.remove("dragging");

	try {
		let dt = e.dataTransfer;
		let files = dt.files;
		handleFile(files[0]);
	} catch (e) {
		logImportError(e.message);
	}
}, false);

// File input
const fileInputElement = document.getElementById("importFile");
fileInputElement.addEventListener("change", () => {
	try {
		let selectedFile = fileInputElement.files[0];
		handleFile(selectedFile);
	} catch (e) {
		logImportError(e.message);
	}
}, false);

// Text input
const textAreaElement = document.getElementById("importText");
const submitButtonElement = document.getElementById("importButton");
submitButtonElement.addEventListener("click", () => {
	try {
		let data = textAreaElement.value;
		if (data.trim().length > 0) {
			handleString(data);
		} else {
			logImportError("Manifest.json text cannot be empty! Please type something in.");
		}
	} catch (e) {
		logImportError(e.message);
	}
}, false);