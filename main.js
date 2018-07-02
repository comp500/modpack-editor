const errorElement = document.getElementById("error");

const textInputMapping = {
	"name": "Pack Name",
	"version": "Version",
	"author": "Author",
	"projectID": "Curse Project ID",
	"overrides": "Overrides folder name"
};

let currentData;

function updateOutput() {
	const exportText = document.getElementById("exportText");
	exportText.value = JSON.stringify(currentData, null, 2);
}

function handleString(input) {
	currentData = JSON.parse(input);
	let inputHandler = key => {
		return (e) => {
			currentData[key] = e.target.value;
			updateOutput();
		};
	};

	let numberInputHandler = key => {
		return (e) => {
			currentData[key] = parseInt(e.target.value);
			updateOutput();
		};
	};

	const output = document.getElementById("output");
	output.classList.remove("d-none");
	updateOutput();

	const editor = document.getElementById("editor");
	hyperHTML.bind(editor)`
	<form>
		${
			Object.keys(textInputMapping).map((key) => {
				let value = currentData[key];
				if (value == null) {
					throw new Error("Key " + key + " doesn't exist in manifest.");
				}

				let handler = inputHandler(key);
				if (isFinite(value)) { // Is it a number?
					handler = numberInputHandler(key);
				}

				return hyperHTML.wire(textInputMapping, ":" + key)`
				<div class="form-group row">
					<label class="col-sm-3 col-form-label" for="${key + "-input"}">${textInputMapping[key]}</label>
					<div class="col-sm-9">
						<input type="text" class="form-control" id="${key + "-input"}" value="${value}" oninput="${handler}">
					</div>
				</div>`;
			})
		}
	</form>
	`;

	errorElement.innerHTML = "";

	//dropbox.classList.add("d-none");
}

/* Import files
*/

function logImportError(message) {
	errorElement.innerText = "Error while importing: " + message;
	console.error(message);
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