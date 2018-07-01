const textInputMapping = {
	"name": "Pack Name",
	"version": "Version",
	"author": "Author",
	"projectID": "Curse Project ID",
	"overrides": "Overrides folder name"
};

let textInputElements = {};

function handleString(input) {
	const inputJSON = JSON.parse(input);
	const editor = document.getElementById("editor");
	editor.innerHTML = null;
	const form = document.createElement("form");

	Object.keys(textInputMapping).forEach((key) => {
		let value = inputJSON[key];
		if (value == null) {
			throw new Error("Key " + key + " doesn't exist in manifest.");
		}

		let formGroup = document.createElement("div");
		formGroup.setAttribute("class", "form-group row");

		let label = document.createElement("label");
		label.appendChild(document.createTextNode(textInputMapping[key]));
		label.setAttribute("class", "col-sm-3 col-form-label");
		label.setAttribute("for", key + "-input");
		formGroup.appendChild(label);

		let inputDiv = document.createElement("div");
		inputDiv.setAttribute("class", "col-sm-9");
		let input = document.createElement("input");
		input.setAttribute("type", "text");
		input.setAttribute("class", "form-control");
		input.setAttribute("id", key + "-input");
		input.value = value;
		inputDiv.appendChild(input);
		formGroup.appendChild(inputDiv);

		form.appendChild(formGroup);

		textInputElements[key] = input;
	});

	editor.appendChild(form);

	//dropbox.classList.add("d-none");
}

/* Import files
*/

const errorElement = document.getElementById("error");
function logImportError(message) {
	errorElement.innerText = "Error while importing: " + message;
}

// Read string data from file
function handleFile(file) {
	let reader = new FileReader();

    reader.onload = function(e) {
		let text = reader.result;
		try {
			handleString(text);
		} catch (e) {
			logImportError(e.message);
		}
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