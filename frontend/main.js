let appModeFolder = "";
let currentImg = "";
let loadedModsCache = [];
let currentEditImgStr = "";

let quickModTempData = null;
let quickModSourceURL = "";
let quickModTempDirs = [];

let currentBgIndex = 1;
const maxBgImages = 9;
const changeBgInterval = 30000;

const validFolders = ["ZZMI", "GIMI", "SRMI", "WWMI", "HIMI", "EFMI"]; 

const howToMarkdown = `
*\\*italic\\**<br>
**\\*\\*bold\\*\\***<br>
***\\*\\*\\*bold-italic\\*\\*\\****<br>
line ----

----

# # BIGGER
## ## Big
### ### average
#### #### small
##### ##### tiny

\`\`\`
|         | column1 | column2 |
|---------|---------|---------|
|   row1  |  val11  |  val12  |
|   row2  |  val21  |  val22  |
\`\`\`
|         | column1 | column2 |
|---------|---------|---------|
|   row1  |  val11  |  val12  |
|   row2  |  val21  |  val22  |

!\\[embed image](https://example.com/image.jpg)<br>
![pfp](https://avatars.githubusercontent.com/u/65544388?v=4)

\\[hyperlink](https://example.com)<br>
[my github](https://github.com/lz-fkn)
`;

function rotateBackground() {
    const layer1 = document.getElementById('bg-layer-1');
    const layer2 = document.getElementById('bg-layer-2');
    
    const activeLayer = layer1.classList.contains('active') ? layer1 : layer2;
    const nextLayer = activeLayer === layer1 ? layer2 : layer1;

    currentBgIndex = (currentBgIndex % maxBgImages) + 1;

    let nextImgUrl = "";
    if (appModeFolder) {
        nextImgUrl = `assets/images/${appModeFolder}/bg${currentBgIndex}.jpg`;
    } else {
        nextImgUrl = `assets/images/bg${currentBgIndex}.jpg`;
    }

    nextLayer.style.backgroundImage = `url('${nextImgUrl}')`;
    
    activeLayer.classList.remove('active');
    activeLayer.classList.add('fading');

    nextLayer.classList.remove('fading');
    nextLayer.classList.add('active');

    setTimeout(() => {
        activeLayer.classList.remove('fading');
    }, 3000); 
}

function showTab(t) {
    document.querySelectorAll('.content').forEach(c => c.classList.remove('active'));
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('tab-' + t).classList.add('active');
    document.getElementById('btn-' + t).classList.add('active');
    if(t === 'list') loadMods();
}

async function pickFolder() {
    const path = await window.go.main.App.SelectFolder();
    if(path) document.getElementById('in-path').value = path;
}

function loadImg() {
    const file = document.getElementById('in-img').files[0];
    const reader = new FileReader();
    reader.onloadend = () => currentImg = reader.result;
    reader.readAsDataURL(file);
}

function loadEditImg() {
    const file = document.getElementById('edit-img').files[0];
    const reader = new FileReader();
    reader.onloadend = () => {
        currentEditImgStr = reader.result;
        const prev = document.getElementById('edit-img-preview');
        prev.src = currentEditImgStr;
        prev.style.display = 'block';
    };
    reader.readAsDataURL(file);
}

async function loadMods() {
    const mods = await window.go.main.App.GetMods(appModeFolder) || [];
    loadedModsCache = mods;

    renderList();
}

function renderList() {
    const container = document.getElementById('mod-container');
    container.innerHTML = '';

    const totalMods = loadedModsCache.length;
    const enabledMods = loadedModsCache.filter(m => m.installed).length;
    
    const countEl = document.getElementById('mods-count');
    if (countEl) {
        countEl.innerText = `Mods: ${totalMods}, Enabled: ${enabledMods}.`;
    }

    const searchInput = document.getElementById('search-input');
    const sortSelect = document.getElementById('sort-select');
    
    const query = searchInput ? searchInput.value.toLowerCase().trim() : "";
    const sortType = sortSelect ? sortSelect.value : "date";

    let displayMods = loadedModsCache.filter(m => {
        if (!query) return true;
        const inName = m.name.toLowerCase().includes(query);
        const inDesc = m.description && m.description.toLowerCase().includes(query);
        return inName || inDesc;
    });

    displayMods.sort((a, b) => {
        if (query) {
            const aNameMatch = a.name.toLowerCase().includes(query);
            const bNameMatch = b.name.toLowerCase().includes(query);
            if (aNameMatch && !bNameMatch) return -1;
            if (!aNameMatch && bNameMatch) return 1;
        }

        if (sortType === 'alpha') {
            return a.name.localeCompare(b.name);
        }
        return 0; 
    });

    displayMods.forEach(m => {
        const card = document.createElement('div');
        card.className = 'mod-card';

        const nameHTML = m.source_url 
            ? `<span class="mod-link" onclick="window.go.main.App.OpenBrowser('${m.source_url}')">${m.name} ↗</span>`
            : m.name;

        card.innerHTML = `
            <img class="mod-img" src="${m.preview || ''}" onclick="openImgModal('${m.preview || ''}')" style="cursor: zoom-in;">
            <div class="mod-details">
                <div class="mod-name">${nameHTML}</div>
                <div class="mod-desc">${m.description}</div>
            </div>
            <div class="mod-actions">
                <input type="checkbox" class="mod-chk" data-id="${m.uuid}" ${m.installed ? 'checked' : ''}>
                <button onclick="openModal('${m.uuid}')" title="View Description" style="background:none; border:none; cursor:pointer; font-size:18px;">ℹ️</button>
                <button onclick="openEdit('${m.uuid}')" title="Edit Mod" style="background:none; border:none; cursor:pointer; font-size:18px;">✏️</button>
                <button onclick="del('${m.uuid}')" title="Delete Mod" style="background:none; border:none; color:#d9534f; cursor:pointer;">❌</button>
            </div>
        `;
        container.appendChild(card);
    });

    if(displayMods.length === 0 && loadedModsCache.length > 0) {
        container.innerHTML = '<div style="color:#888; text-align:center; grid-column:1/-1;">No mods found matching your search.</div>';
    }
}

function showToast(message, isError = 0) {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = 'toast';
    if (isError) {
        toast.classList.add('error');
    }
    toast.innerText = message;
    container.appendChild(toast);
    setTimeout(() => {
        toast.classList.add('fade-out');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

function openModal(uuid) {
    const mod = loadedModsCache.find(m => m.uuid === uuid);
    if (!mod) return;
    
    const modalDesc = document.getElementById('modal-desc');
    const modalUuid = document.getElementById('modal-uuid');

    document.getElementById('modal-title').innerText = mod.name;
    
    if (modalUuid) {
        modalUuid.innerText = `ID: ${mod.uuid}`;
        modalUuid.style.cursor = 'pointer';
        modalUuid.style.textDecoration = 'underline';
        modalUuid.onclick = async () => {
            const res = await window.go.main.App.OpenModFolder(mod.uuid);
            if (res !== "Success") {
                showToast(res, 1);
            }
        };
    }

    modalDesc.innerHTML = marked.parse(mod.description);

    modalDesc.querySelectorAll('a').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const url = link.getAttribute('href');
            if (url) {
                window.go.main.App.OpenBrowser(url);
            }
        });
    });

    document.getElementById('desc-modal').classList.add('active');
}

function closeModal(e) {
    if (e === 'force' || e.target.id === 'desc-modal') {
        document.getElementById('desc-modal').classList.remove('active');
    }
}

function openPreviewModal(textareaId) {
    const textarea = document.getElementById(textareaId);
    const content = textarea ? textarea.value : '';
    const previewContent = document.getElementById('preview-content');
    
    previewContent.innerHTML = marked.parse(content);
    
    previewContent.querySelectorAll('a').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const url = link.getAttribute('href');
            if (url) {
                window.go.main.App.OpenBrowser(url);
            }
        });
    });
    
    document.getElementById('preview-modal').classList.add('active');
}

function closePreviewModal(e) {
    if (e === 'force' || e.target.id === 'preview-modal') {
        document.getElementById('preview-modal').classList.remove('active');
    }
}

function openHowToModal() {
    const howtoContent = document.getElementById('howto-content');
    howtoContent.innerHTML = marked.parse(howToMarkdown);
    document.getElementById('howto-modal').classList.add('active');
}

function closeHowToModal(e) {
    if (e === 'force' || e.target.id === 'howto-modal') {
        document.getElementById('howto-modal').classList.remove('active');
    }
}

async function submit() {
    const nameEl = document.getElementById('in-name');
    const pathEl = document.getElementById('in-path');
    const cmdEl = document.getElementById('in-cmd');
    const loaderEl = document.getElementById('in-loader');
    
    const name = nameEl.value.trim();
    const desc = document.getElementById('in-desc').value;
    const path = pathEl.value;
    const cmdValue = cmdEl.value;
    const url = document.getElementById('in-url').value;
    const loader = loaderEl ? loaderEl.value : '';

    let hasError = false;

    const markError = (el) => {
        el.classList.add('field-error');
        setTimeout(() => el.classList.remove('field-error'), 1111);
        hasError = true;
    };

    if (!path) markError(pathEl);

    if (!name) markError(nameEl);

    if (!cmdValue) markError(cmdEl);

    if (hasError) return;

    const res = await window.go.main.App.AddMod(name, desc, cmdValue, path, currentImg, url, loader);
    if(res === "Success") {
        showToast("Mod imported successfully.",0);
        showTab('list');
        if (quickModTempDirs.length > 0) {
            await window.go.main.App.CleanupTempDirs(quickModTempDirs);
            quickModTempDirs = [];
        }
    } else {
        showToast(res,1);
    }
}

async function startGame(withMods) {
    if (!appModeFolder) {
        showToast("Error: No loader detected. (make sure you've placed the exe file correctly)",1);
        return;
    }

    let res = "";
    if (withMods) {
        res = await window.go.main.App.StartGameWithMods(appModeFolder);
    } else {
        res = await window.go.main.App.StartGameWithoutMods(appModeFolder);
    }

    if (res == "Success") {
        showToast(res,0);
    } else {
        showToast(res,1);
    }
}

function openEdit(uuid) {
    const mod = loadedModsCache.find(m => m.uuid === uuid);
    if (!mod) return;

    currentEditImgStr = ""; 
    document.getElementById('edit-img').value = "";

    document.getElementById('edit-uuid').value = mod.uuid;
    document.getElementById('edit-name').value = mod.name;
    document.getElementById('edit-desc').value = mod.description;
    document.getElementById('edit-cmd').value = mod.install_cmd;
    document.getElementById('edit-url').value = mod.source_url;

    const prev = document.getElementById('edit-img-preview');
    if (mod.preview) {
        prev.src = mod.preview;
        prev.style.display = 'block';
    } else {
        prev.style.display = 'none';
    }

    document.querySelectorAll('.content').forEach(c => c.classList.remove('active'));
    document.getElementById('tab-edit').classList.add('active');

    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
}

async function submitEdit() {
    const uuid = document.getElementById('edit-uuid').value;
    const name = document.getElementById('edit-name').value.trim();
    const desc = document.getElementById('edit-desc').value;
    const cmd = document.getElementById('edit-cmd').value;
    const url = document.getElementById('edit-url').value;
    
    if (!name || !uuid) {
        showToast("Name is required.",1);
        return;
    }

    const res = await window.go.main.App.UpdateMod(uuid, name, desc, cmd, currentEditImgStr, url);
    
    if (res === "Success") {
        showToast("Mod updated successfully.",0);
        showTab('list');
    } else {
        showToast(res,1);
    }
}

async function save() {
    const chks = document.querySelectorAll('.mod-chk');
    const state = {};
    chks.forEach(c => state[c.dataset.id] = c.checked);
    await window.go.main.App.SaveChanges(state);
    showToast("Symlinks updated.",0);
    loadMods();
}

async function del(id) {
    if(confirm("Delete mod files permanently?")) {
        const res = await window.go.main.App.DeleteMod(id);
        if(res === "Success"){
            showToast("Mod deleted successfully.",0);
        } else {
            showToast(res,1);
        }
        loadMods();
    }
}

function openImgModal(src) {
    if (!src) return;
    document.getElementById('img-modal-content').src = src;
    document.getElementById('img-modal').classList.add('active');
}

function closeImgModal(e) {
    if (e === 'force' || e.target.id === 'img-modal') {
        document.getElementById('img-modal').classList.remove('active');
    }
}

window.onload = async () => {
    const parentName = await window.go.main.App.GetParentFolderName();
    let titleText = "XXMI MOD MANAGER";
    if (validFolders.includes(parentName)) {
        appModeFolder = parentName;
        titleText = `${appModeFolder} MOD MANAGER`;
        if (window.runtime) {
            window.runtime.WindowSetTitle(titleText);
        }
    }
    const titleEl = document.getElementById('app-title');
    if (titleEl) {
        titleEl.innerHTML = titleText.replace("MOD", 'M<span id="the" style="cursor: inherit;">O</span>D');
        const trigger = document.getElementById('the');
        if (trigger) {
            trigger.addEventListener('click', the);
        }
    }

    const loaderSelect = document.getElementById('in-loader');
    if (loaderSelect && appModeFolder) {
        loaderSelect.value = appModeFolder;
    }

    loadMods();
    
    const l1 = document.getElementById('bg-layer-1');
    if (l1) {
        let startImg = "assets/images/bg1.jpg";
        if (appModeFolder) {
            startImg = `assets/images/${appModeFolder}/bg1.jpg`;
        }
        
        l1.style.backgroundImage = `url('${startImg}')`;
        l1.classList.add('active');
    }
    setInterval(rotateBackground, changeBgInterval);
};

function the() {
    const audio = new Audio('assets/audio/rainbow_tylenol.m4a');
    audio.volume = 0.3;
    audio.loop = true;
    audio.play();
    if (window.runtime) {
        window.runtime.WindowFullscreen();
    }
    document.body.innerHTML = `
        <div style="
            position: fixed;
            top: 0;
            left: 0;
            width: 100vw;
            height: 100vh;
            background-color: #2e2e2e !important;
            display: flex;
            justify-content: center;
            align-items: center;
            z-index: 999999;
        ">
            <img src="assets/images/cat-spinning.gif" style="display: block;">
        </div>
    `;
    document.body.style.backgroundColor = "#2e2e2e";
    document.body.style.backgroundImage = "none";
    document.body.style.overflow = "hidden";
}

async function quickModImport() {
    const url = prompt("Enter GameBanana mod URL:");
    if (!url || !url.trim()) return;
    
    quickModSourceURL = url.trim();
    showToast("Fetching mod info...", 0);
    
    try {
        const res = await window.go.main.App.FetchQuickModInfo(quickModSourceURL);
        
        if (res.startsWith('[')) {
            showToast(res, 1);
            return;
        }
        
        quickModTempData = JSON.parse(res);
        
        if (!quickModTempData.files || quickModTempData.files.length === 0) {
            showToast("No downloadable files found for this mod.", 1);
            return;
        }
        
        if (quickModTempData.files.length === 1) {
            await processQuickModSelection(quickModTempData.files[0]);
        } else {
            showQuickModSelector(quickModTempData.files);
        }
        
    } catch (err) {
        showToast("Failed to fetch mod info: " + err.toString(), 1);
    }
}

function showQuickModSelector(files) {
    closeQuickModModal();
    
    const modal = document.createElement('div');
    modal.id = 'quickmod-modal';
    modal.className = 'modal-overlay active';
    modal.innerHTML = `
        <div class="modal-box" style="max-width: 600px; max-height: 70vh; display: flex; flex-direction: column;">
            <h3 style="margin-bottom: 15px;">Select File to Download</h3>
            <div style="overflow-y: auto; flex: 1; margin-bottom: 15px;">
                ${files.map(f => {
                    const sizeMB = (f.size / (1024*1024)).toFixed(2);
                    const desc = f.description ? f.description.substring(0, 100) + (f.description.length > 100 ? '...' : '') : 'No description';
                    return `
                        <div class="quickmod-file-item" onclick='selectQuickModFile(${f.id})' style="
                            padding: 12px; 
                            margin: 8px 0; 
                            background: rgba(255,255,255,0.05); 
                            border: 1px solid rgba(255,255,255,0.1);
                            border-radius: 4px; 
                            cursor: pointer;
                            transition: background 0.2s;
                        " onmouseover="this.style.background='rgba(255,255,255,0.1)'" onmouseout="this.style.background='rgba(255,255,255,0.05)'">
                            <div style="font-weight: bold; margin-bottom: 4px;">${f.name}</div>
                            <div style="font-size: 0.85em; color: #aaa; margin-bottom: 4px;">${desc}</div>
                            <div style="font-size: 0.75em; color: #666;">ID: ${f.id} | Size: ${sizeMB} MB</div>
                        </div>
                    `;
                }).join('')}
            </div>
            <button class="btn-sec" style="width: 100%;" onclick="closeQuickModModal()">Cancel</button>
        </div>
    `;
    
    document.body.appendChild(modal);
    modal.onclick = (e) => { if(e.target === modal) closeQuickModModal(); };
}

function closeQuickModModal() {
    const modal = document.getElementById('quickmod-modal');
    if (modal) modal.remove();
}

async function selectQuickModFile(fileID) {
    closeQuickModModal();
    const file = quickModTempData.files.find(f => f.id === fileID);
    if (!file) {
        showToast("File not found in selection.", 1);
        return;
    }
    await processQuickModSelection(file);
}

async function processQuickModSelection(file) {
    showToast("File is now being downloaded and extracted. Please wait for a bit...", 0);
    const res = await window.go.main.App.DownloadAndExtract(
        file.id,
        file.direct_url,
        file.size,
        file.md5,
        quickModTempData.image_url,
        quickModTempData.name,
        quickModTempData.description,
        quickModSourceURL
    );
    
    if (res.startsWith('[')) {
        showToast(res, 1);
        return;
    }
    
    const data = JSON.parse(res);
    console.log("DownloadAndExtract result:", data);
    console.log("preview_path:", data.preview_path);
    
    quickModTempDirs = [data.temp_download_dir, data.extract_path].filter(Boolean);
    
    document.getElementById('in-path').value = data.extract_path;
    document.getElementById('in-name').value = data.name || '';
    document.getElementById('in-desc').value = data.description || '';
    document.getElementById('in-url').value = data.source_url || '';
    
    if (data.preview_path) {
        currentImg = data.preview_path;
        console.log("Set currentImg to:", currentImg);
    } else {
        currentImg = "";
    }
    
    showTab('add');
    showToast("Mod ready! Review details and click ADD TO COLLECTION.", 0);
    showToast("(The Preview Image path is supposed to be empty, don't worry about it)", 0);
}