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

function renderForm() {
	let inputHandler = key => {
		return e => {
			currentData[key] = e.target.value;
		};
	};

	let numberInputHandler = key => {
		return e => {
			currentData[key] = parseInt(e.target.value);
		};
	};

	let mcVersionInputHandler = e => {
		currentData.minecraft.version = e.target.value;
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
	};

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
}

function handleString(input) {
	currentData = JSON.parse(input);
	renderForm();
}

const openMessagesElement = document.getElementById("openMessages");

function logOpenError(message) {
	openMessagesElement.innerText = "Error while opening: " + message;
	openMessagesElement.className = "text-danger";
	console.error(message);
}

function showOpenSuccess(isCreated) {
	openMessagesElement.innerText = isCreated ? "Successfully created modpack!" : "Successfully opened modpack!";
	openMessagesElement.className = "text-success";
}

const modpackLocationInput = document.getElementById("modpackLocation");
const openModpackButtonElement = document.getElementById("openModpackButton");
openModpackButtonElement.addEventListener("click", () => {
	fetch("/ajax/loadModpackFolder", {
		method: "post",
		headers: {
			"Content-type": "application/json; charset=UTF-8"
		},
		body: JSON.stringify({
			"Folder": modpackLocationInput.value
		})
	}).then(response => response.json()).then(function(data) {
		if (data.ErrorMessage) {
			logOpenError(data.ErrorMessage);
			return;
		}
		showOpenSuccess(false);
		currentData = data.Modpack.CurseManifest;
		renderForm();
	}).catch(function(error) {
		logOpenError(error);
	});
}, false);

const newModpackButtonElement = document.getElementById("newModpackButton");
newModpackButtonElement.addEventListener("click", () => {
	fetch("/ajax/createModpackFolder", {
		method: "post",
		headers: {
			"Content-type": "application/json; charset=UTF-8"
		},
		body: JSON.stringify({
			"Folder": modpackLocationInput.value
		})
	}).then(response => response.json()).then(function(data) {
		if (data.ErrorMessage) {
			logOpenError(data.ErrorMessage);
			return;
		}
		showOpenSuccess(true);
		currentData = data.Modpack.CurseManifest;
		renderForm();
	}).catch(function(error) {
		logOpenError(error);
	});
}, false);