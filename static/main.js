let currentModpack;

// TODO: support SSC description?
// TODO: support custom server config of mcVersion and forge?

// Go equates [] and null, JavaScript does not.
function nullableArray(array) {
	if (array == null) {
		return [];
	} else {
		return array;
	}
}

const generalInputHandlers = [
	{
		id: "name",
		label: "Pack Name",
		handler: e => {
			currentModpack.CurseManifest.name = e.target.value;
			currentModpack.ServerSetupConfig.Name = e.target.value;
		},
		get: () => currentModpack.CurseManifest.name
	},
	{
		id: "version",
		label: "Version",
		handler: e => currentModpack.CurseManifest.version = e.target.value,
		get: () => currentModpack.CurseManifest.version
	},
	{
		id: "author",
		label: "Author",
		handler: e => currentModpack.CurseManifest.author = e.target.value,
		get: () => currentModpack.CurseManifest.author
	},
	{
		id: "projectID",
		label: "Curse Project ID",
		handler: e => currentModpack.CurseManifest.projectID = parseInt(e.target.value),
		get: () => currentModpack.CurseManifest.projectID
	},
	{
		id: "overrides",
		label: "Overrides folder name",
		handler: e => currentModpack.CurseManifest.overrides = e.target.value,
		get: () => currentModpack.CurseManifest.overrides
	},
	{
		id: "mcVersion",
		label: "Minecraft version",
		handler: e => currentModpack.CurseManifest.minecraft.version = e.target.value,
		get: () => currentModpack.CurseManifest.minecraft.version
	},
	{
		id: "modLoaders",
		label: "Modloader ID(s) (e.g. forge-14.23.4.2715)",
		handler: e => {
			let isFirst = true;
			currentModpack.CurseManifest.minecraft.modLoaders = e.target.value.split(",").map(modLoader => {
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
			let value = nullableArray(currentModpack.CurseManifest.minecraft.modLoaders);

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

//currentModpack.ServerSetupConfig.Install.FormatSpecific.IgnoreProject
const serverInputHandlers = [
	{
		id: "packURL",
		label: "Pack download link",
		handler: e => currentModpack.ServerSetupConfig.Install.ModpackURL = e.target.value,
		get: () => currentModpack.ServerSetupConfig.Install.ModpackURL
	},
	// TODO: IgnoreProject (with mod list?)
	{
		id: "baseInstallPath",
		label: "Installation path (leave empty for current path)",
		handler: e => currentModpack.ServerSetupConfig.Install.BaseInstallPath = e.target.value,
		get: () => currentModpack.ServerSetupConfig.Install.BaseInstallPath
	},
	{
		id: "ignoreFiles",
		label: "Folders to ignore",
		handler: e => currentModpack.ServerSetupConfig.Install.IgnoreFiles = e.target.value.split(",").map(a => a.trim()),
		get: () => nullableArray(currentModpack.ServerSetupConfig.Install.IgnoreFiles).join(",")
	},
	// TODO: additionalFiles (with mod list?)
	// TODO: localFiles
	{
		id: "checkFolder",
		label: "Check folder (?)",
		type: "checkbox",
		handler: e => currentModpack.ServerSetupConfig.Install.CheckFolder = e.target.checked,
		get: () => currentModpack.ServerSetupConfig.Install.CheckFolder
	},
	{
		id: "installForge",
		label: "Install Forge",
		type: "checkbox",
		handler: e => currentModpack.ServerSetupConfig.Install.InstallForge = e.target.checked,
		get: () => currentModpack.ServerSetupConfig.Install.InstallForge
	},
	{
		id: "spongeFix",
		label: "Apply launch wrapper to fix Sponge",
		type: "checkbox",
		handler: e => currentModpack.ServerSetupConfig.Launch.SpongeFix = e.target.checked,
		get: () => currentModpack.ServerSetupConfig.Launch.SpongeFix
	},
	{
		id: "checkOffline",
		label: "Check if server is online while installing",
		type: "checkbox",
		handler: e => currentModpack.ServerSetupConfig.Launch.CheckOffline = e.target.checked,
		get: () => currentModpack.ServerSetupConfig.Launch.CheckOffline
	},
	{
		id: "maxRAM",
		label: "Maximum RAM allocation",
		handler: e => currentModpack.ServerSetupConfig.Launch.MaxRAM = e.target.value,
		get: () => currentModpack.ServerSetupConfig.Launch.MaxRAM
	},
	{
		id: "autoRestart",
		label: "Auto restart server after crash",
		type: "checkbox",
		handler: e => currentModpack.ServerSetupConfig.Launch.AutoRestart = e.target.checked,
		get: () => currentModpack.ServerSetupConfig.Launch.AutoRestart
	},
	{
		id: "crashLimit",
		label: "Number of crashes to stop restarting after",
		handler: e => currentModpack.ServerSetupConfig.Launch.CrashLimit = parseInt(e.target.value),
		get: () => currentModpack.ServerSetupConfig.Launch.CrashLimit
	},
	{
		id: "crashTimer",
		label: "Time to count crash number within",
		handler: e => currentModpack.ServerSetupConfig.Launch.CrashTimer = e.target.value,
		get: () => currentModpack.ServerSetupConfig.Launch.CrashTimer
	},
	{
		id: "preJavaArgs",
		label: "Arguments before java command",
		handler: e => currentModpack.ServerSetupConfig.Launch.PreJavaArgs = e.target.value,
		get: () => currentModpack.ServerSetupConfig.Launch.PreJavaArgs
	},
	// TODO: make java args easier to edit
	{
		id: "javaArgs",
		label: "Java arguments",
		handler: e => currentModpack.ServerSetupConfig.Launch.JavaArgs = e.target.value.split(",").map(a => a.trim()),
		get: () => nullableArray(currentModpack.ServerSetupConfig.Launch.JavaArgs).join(",")
	}
];

function renderForm() {
	let inputHandlerWire = (handlers) => handlers.map(inputObject => {
		let value = inputObject.get();

		// may be falsy, so use ===
		if (value === null || value === undefined) {
			throw new Error("Key " + inputObject.id + " doesn't exist in manifest.");
		}

		if (inputObject.type == "checkbox") {
			return hyperHTML.wire(currentModpack, ":" + inputObject.id)`
			<div class="form-group row">
				<div class="col-sm-3"></div>
				<div class="col-sm-9">
					<div class="form-check">
						<input class="form-check-input" type="checkbox" id="${inputObject.id + "-input"}" checked="${value}" oninput="${inputObject.handler}">
						<label class="form-check-label" for="${inputObject.id + "-input"}">${inputObject.label}</label>
					</div>
				</div>
			</div>
			`;
		}

		return hyperHTML.wire(currentModpack, ":" + inputObject.id)`
		<div class="form-group row">
			<label class="col-sm-3 col-form-label" for="${inputObject.id + "-input"}">${inputObject.label}</label>
			<div class="col-sm-9">
				<input type="${inputObject.type ? inputObject.type : "text"}" class="form-control" id="${inputObject.id + "-input"}" value="${value}" oninput="${inputObject.handler}">
			</div>
		</div>`;
	});

	const generalSettings = document.getElementById("generalSettings");
	hyperHTML.bind(generalSettings)`${inputHandlerWire(generalInputHandlers)}`;

	const serverSettings = document.getElementById("serverSettings");
	hyperHTML.bind(serverSettings)`${inputHandlerWire(serverInputHandlers)}`;

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

				modListLink.innerText = "Mod list (" + currentModpack.CurseManifest.files.length + " mods)";

				return currentModpack.CurseManifest.files.sort((a, b) => {
					// Push missing projects to the top
					if (!data[a.projectID] || data[a.projectID].ErrorMessage) {
						return -1;
					} else if (!data[b.projectID] || data[b.projectID].ErrorMessage) {
						return 1;
					}
					return data[a.projectID].Name.localeCompare(data[b.projectID].Name);
				}).map(currentMod => {
					let currentModData = data[currentMod.projectID];
					if (!currentModData || currentModData.ErrorMessage) {
						return hyperHTML.wire()`
						<li class="list-group-item list-group-item-warning flex-row d-flex">
							<img src="/MissingTexture.png" class="img-thumbnail modIcon mr-2">
							<div class="flex-fill">
								<h5 class="mb-1">An error occurred (project id ${currentMod.projectID})</h5>
								<p class="mb-1">${currentModData ? currentModData.ErrorMessage : ""}</p>
							</div>
						</li>
						`;
					}

					let iconURL = currentModData.IconURL ? currentModData.IconURL : "/MissingTexture.png";
					// Replace curseforge with minecraft.curseforge
					let websiteURL = currentModData.WebsiteURL.replace("www.curseforge.com/minecraft/mc-mods/", "minecraft.curseforge.com/projects/");

					return hyperHTML.wire()`
					<li class="list-group-item flex-row d-flex">
						<img src="${iconURL}" class="img-thumbnail modIcon mr-2">
						<div class="flex-fill">
							<div class="d-flex justify-content-between">
								<h5 class="mb-1"><a href="${websiteURL}">${currentModData.Name}</a></h5>
								<small class="text-muted">3 days ago</small>
							</div>
							<p class="mb-1">${currentModData.Summary}</p>
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
		currentModpack = data.Modpack;
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
		currentModpack = data.Modpack;
		renderForm();
	}).catch(function(error) {
		logOpenError(error);
	});
}, false);

const reloadModpackButtonElement = document.getElementById("reloadModpackButton");
reloadModpackButtonElement.addEventListener("click", () => {
	if (!currentModpack) {
		return;
	}
	modpackLocationInput.value = currentModpack.Folder;
	fetch("/ajax/loadModpackFolder", {
		method: "post",
		headers: {
			"Content-type": "application/json; charset=UTF-8"
		},
		body: JSON.stringify({
			"Folder": currentModpack.Folder
		})
	}).then(response => response.json()).then(function(data) {
		if (data.ErrorMessage) {
			logOpenError(data.ErrorMessage);
			return;
		}
		showOpenSuccess(false);
		currentModpack = data.Modpack;
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
	currentModpack = data.Modpack;
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