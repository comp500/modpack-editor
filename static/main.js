const errorElement = document.getElementById("error");

const textInputMapping = {
	"name": "Pack Name",
	"version": "Version",
	"author": "Author",
	"projectID": "Curse Project ID",
	"overrides": "Overrides folder name"
};

const blankTemplate = {
    "minecraft": {
        "version": "",
        "modLoaders": []
    },
    "manifestType": "minecraftModpack",
    "manifestVersion": 1,
    "name": "",
    "version": "",
    "author": "",
    "projectID": 0,
    "files": [],
    "overrides": "overrides"
};

let currentData;

function updateOutput() {
	const exportText = document.getElementById("exportText");
	exportText.value = JSON.stringify(currentData, null, 2);
}

function renderForm() {
	let inputHandler = key => {
		return e => {
			currentData[key] = e.target.value;
			updateOutput();
		};
	};

	let numberInputHandler = key => {
		return e => {
			currentData[key] = parseInt(e.target.value);
			updateOutput();
		};
	};

	let mcVersionInputHandler = e => {
		currentData.minecraft.version = e.target.value;
		updateOutput();
	};

	// Split list into objects, with primary modLoader first
	let modLoaderInputHandler = e => {
		let isFirst = true;
		currentData.minecraft.modLoaders = e.target.value.split(",").map(modLoader => {
			if (isFirst) {
				isFirst = false;
				return {
					id: modLoader.trim(),
					primary: true
				};
			}
			return {
				id: modLoader.trim()
			};
		});
		updateOutput();
	};

	const output = document.getElementById("output");
	output.classList.remove("d-none");
	updateOutput();

	const editor = document.getElementById("editor");
	hyperHTML.bind(editor)`
	<h3>Edit modpack</h3>
	<form>
		${
			Object.keys(textInputMapping).map((key) => {
				let value = currentData[key];
				if (value == null) {
					throw new Error("Key " + key + " doesn't exist in manifest.");
				}

				let handler;
				if (Number.isInteger(value)) { // Is it a number?
					handler = numberInputHandler(key);
				} else {
					handler = inputHandler(key);
				}

				return hyperHTML.wire(currentData, ":" + key)`
				<div class="form-group row">
					<label class="col-sm-3 col-form-label" for="${key + "-input"}">${textInputMapping[key]}</label>
					<div class="col-sm-9">
						<input type="text" class="form-control" id="${key + "-input"}" value="${value}" oninput="${handler}">
					</div>
				</div>`;
			})
		}
		${
			(() => {
				let value = currentData.minecraft.version;
				if (value == null) {
					throw new Error("Minecraft version doesn't exist in manifest.");
				}

				return hyperHTML.wire(currentData, ":mcVersion")`
				<div class="form-group row">
					<label class="col-sm-3 col-form-label" for="mcVersion-input">Minecraft version</label>
					<div class="col-sm-9">
						<input type="text" class="form-control" id="mcVersion-input" value="${value}" oninput="${mcVersionInputHandler}">
					</div>
				</div>`;
			})()
		}
		${
			(() => {
				let value = currentData.minecraft.modLoaders;
				if (value == null) {
					throw new Error("Modloaders list doesn't exist in manifest.");
				}

				value.sort((a, b) => {
					// Put the primary value first
					if (a.primary && !b.primary) {
						return -1;
					}
					if (!a.primary && b.primary) {
						return 1;
					}
					return 0;
				});
				let valueConverted = value.map(a => a.id).join(",");

				return hyperHTML.wire(currentData, ":modLoaders")`
				<div class="form-group row">
					<label class="col-sm-3 col-form-label" for="mcVersion-input">Modloader ID(s) (e.g. forge-14.23.4.2715)</label>
					<div class="col-sm-9">
						<input type="text" class="form-control" id="mcVersion-input" value="${valueConverted}" oninput="${modLoaderInputHandler}">
					</div>
				</div>`;
			})()
		}
	</form>
	`;

	errorElement.innerHTML = "";

	//dropbox.classList.add("d-none");
}

function handleString(input) {
	currentData = JSON.parse(input);
	renderForm();
}

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

// Creation from blank
const newButtonElement = document.getElementById("newButton");
newButtonElement.addEventListener("click", () => {
	try {
		data = JSON.stringify(blankTemplate);
		handleString(data);
	} catch (e) {
		logImportError(e.message);
	}
}, false);