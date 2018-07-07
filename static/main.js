const generalInputHandlers = [
	{
		id: "name",
		label: "Pack Name",
		handler: e => currentData.name = e.target.value,
		get: () => currentData.name
	},
	{
		id: "version",
		label: "Version",
		handler: e => currentData.version = e.target.value,
		get: () => currentData.version
	},
	{
		id: "author",
		label: "Author",
		handler: e => currentData.author = e.target.value,
		get: () => currentData.author
	},
	{
		id: "projectID",
		label: "Curse Project ID",
		handler: e => currentData.projectID = parseInt(e.target.value),
		get: () => currentData.projectID
	},
	{
		id: "overrides",
		label: "Overrides folder name",
		handler: e => currentData.overrides = e.target.value,
		get: () => currentData.overrides
	},
	{
		id: "mcVersion",
		label: "Minecraft version",
		handler: e => currentData.minecraft.version = e.target.value,
		get: () => currentData.minecraft.version
	},
	{
		id: "modLoaders",
		label: "Modloader ID(s) (e.g. forge-14.23.4.2715)",
		handler: e => {
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
		},
		get: () => {
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
			return value.map(a => a.id).join(",");
		}
	}
];

let currentData;

function renderForm() {
	const generalSettings = document.getElementById("generalSettings");
	hyperHTML.bind(generalSettings)`
	${
		generalInputHandlers.map(inputObject => {
			let value = inputObject.get();

			if (value == null) {
				throw new Error("Key " + inputObject.id + " doesn't exist in manifest.");
			}
			return hyperHTML.wire(currentData, ":" + inputObject.id)`
			<div class="form-group row">
				<label class="col-sm-3 col-form-label" for="${inputObject.id + "-input"}">${inputObject.label}</label>
				<div class="col-sm-9">
					<input type="text" class="form-control" id="${inputObject.id + "-input"}" value="${value}" oninput="${inputObject.handler}">
				</div>
			</div>`;
		})
	}
	`;

	// Request mod data for each mod
	const modList = document.getElementById("modList");
	const modListLink = document.getElementById("modListLink");
	hyperHTML.bind(modList)`
	<ul class="list-group">
		${{
			any: fetch("/ajax/getModInfoList").then(response => response.json()).then(function(data) {
				if (data.ErrorMessage) {
					logOpenError(data.ErrorMessage);
					return;
				}

				modListLink.innerText = "Mod list (" + currentData.files.length + " mods)";

				return currentData.files.sort((a, b) => {
					// Push missing projects to the top
					if (!data[a.projectID] || data[a.projectID].ErrorMessage) {
						return -1;
					} else if (!data[b.projectID] || data[b.projectID].ErrorMessage) {
						return 1;
					}
					return data[a.projectID].Name.localeCompare(data[b.projectID].Name);
				}).map(currentMod => {
					let currentData = data[currentMod.projectID];
					if (!currentData || currentData.ErrorMessage) {
						return hyperHTML.wire()`
						<li class="list-group-item list-group-item-warning flex-row d-flex">
							<img src="/MissingTexture.png" class="img-thumbnail modIcon mr-2">
							<div class="flex-fill">
								<h5 class="mb-1">An error occurred (project id ${currentMod.projectID})</h5>
								<p class="mb-1">${currentData ? currentData.ErrorMessage : ""}</p>
							</div>
						</li>
						`;
					}

					let iconURL = currentData.IconURL ? currentData.IconURL : "/MissingTexture.png";
					// Replace curseforge with minecraft.curseforge
					let websiteURL = currentData.WebsiteURL.replace("www.curseforge.com/minecraft/mc-mods/", "minecraft.curseforge.com/projects/");

					return hyperHTML.wire()`
					<li class="list-group-item flex-row d-flex">
						<img src="${iconURL}" class="img-thumbnail modIcon mr-2">
						<div class="flex-fill">
							<div class="d-flex justify-content-between">
								<h5 class="mb-1"><a href="${websiteURL}">${currentData.Name}</a></h5>
								<small class="text-muted">3 days ago</small>
							</div>
							<p class="mb-1">${currentData.Summary}</p>
						</div>
					</li>
					`;
				})
			}).catch(function(error) {
				logOpenError(error);
			}),
			placeholder: "Loading mod list..."
		}}
	</ul>
	`;
	// Unhide editor
	const editor = document.getElementById("editor");
	editor.classList.remove("d-none");
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

// Modpack opening UI
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

// Load current modpack
fetch("/ajax/getCurrentPackDetails").then(response => response.json()).then(function(data) {
	if (data.ErrorMessage) {
		logOpenError(data.ErrorMessage);
		return;
	}
	if (data.Modpack == null) { // No modpack loaded yet
		return;
	}
	showOpenSuccess(false);
	modpackLocationInput.value = data.Modpack.Folder;
	currentData = data.Modpack.CurseManifest;
	renderForm();
}).catch(function(error) {
	logOpenError(error);
});

// Tabbed UI
function createTabbedUI(tabs, links) {
	let tabElements = tabs.map((a) => document.getElementById(a));
	let linkElements = links.map((a) => document.getElementById(a));
	linkElements.forEach((linkEl, i) => {
		let currentEl = tabElements[i];
		linkEl.addEventListener("click", e => {
			e.preventDefault();

			currentEl.classList.remove("d-none");
			tabElements.forEach((unsel) => {
				if (unsel != currentEl) {
					unsel.classList.add("d-none"); 
				}
			});
			linkEl.classList.add("active");
			linkElements.forEach((unsel) => {
				if (unsel != linkEl) {
					unsel.classList.remove("active"); 
				}
			});
		}, false);
	});
}

createTabbedUI(["generalSettings", "serverSettings", "modList", "addNewMods"], ["generalSettingsLink", "serverSettingsLink", "modListLink", "addNewModsLink"]);