let currentImg = "";
let loadedModsCache = [];

let currentBgIndex = 1;
const maxBgImages = 9;
const changeBgInterval = 30000;

function rotateBackground() {
    const layer1 = document.getElementById('bg-layer-1');
    const layer2 = document.getElementById('bg-layer-2');
    
    const activeLayer = layer1.classList.contains('active') ? layer1 : layer2;
    const nextLayer = activeLayer === layer1 ? layer2 : layer1;

    currentBgIndex = (currentBgIndex % maxBgImages) + 1;
    const nextImgUrl = `assets/images/bg${currentBgIndex}.jpg`;

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

async function loadMods() {
    const mods = await window.go.main.App.GetMods() || [];
    loadedModsCache = mods;

    renderList();
}

function renderList() {
    const container = document.getElementById('mod-container');
    container.innerHTML = '';

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
            <img class="mod-img" src="${m.preview || ''}">
            <div class="mod-details">
                <div class="mod-name">${nameHTML}</div>
                <div class="mod-desc">${m.description}</div>
            </div>
            <div class="mod-actions">
                <input type="checkbox" class="mod-chk" data-id="${m.uuid}" ${m.installed ? 'checked' : ''}>
                <button onclick="openModal('${m.uuid}')" title="View Description" style="background:none; border:none; cursor:pointer; font-size:18px;">ℹ️</button>
                <button onclick="del('${m.uuid}')" title="Delete Mod" style="background:none; border:none; color:#d9534f; cursor:pointer;">❌</button>
            </div>
        `;
        container.appendChild(card);
    });

    if(displayMods.length === 0 && loadedModsCache.length > 0) {
        container.innerHTML = '<div style="color:#888; text-align:center; grid-column:1/-1;">No mods found matching your search.</div>';
    }
}

function showToast(message) {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = 'toast';
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
    document.getElementById('modal-title').innerText = mod.name;
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

async function submit() {
    const nameEl = document.getElementById('in-name');
    const pathEl = document.getElementById('in-path');
    const cmdEl = document.getElementById('in-cmd');
    
    const name = nameEl.value.trim();
    const desc = document.getElementById('in-desc').value;
    const path = pathEl.value;
    const cmdValue = cmdEl.value;
    const url = document.getElementById('in-url').value;

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

    const res = await window.go.main.App.AddMod(name, desc, cmdValue, path, currentImg, url);
    if(res === "Success") {
        showToast("Mod imported successfully.");
        /* document.getElementById('in-name').value = ''; -- DO NOT CLEAN UP all those for now 
        document.getElementById('in-desc').value = '';
        document.getElementById('in-path').value = '';
        document.getElementById('in-url').value = ''; 
        cmdEl.value = '';
        currentImg = ""; */
        showTab('list');
    } else {
        showToast(res);
    }
}

async function save() {
    const chks = document.querySelectorAll('.mod-chk');
    const state = {};
    chks.forEach(c => state[c.dataset.id] = c.checked);
    await window.go.main.App.SaveChanges(state);
    showToast("Symlinks updated.");
    loadMods();
}

async function del(id) {
    if(confirm("Delete mod files permanently?")) {
        const res = await window.go.main.App.DeleteMod(id);
        if(res === "Success"){
            showToast("Mod deleted successfully.");
        } else {
            showToast(res);
        }
        loadMods();
    }
}

const aboutMarkdown = `
# XXMI Manager
*rather simple mod manager for XXMI mods (ZZZ, GI, HSR, WuWa (i think))*

---

made it when i've realised that i need some easy way to toggle on and off the mods on the fly, so here it is. it works by creating a symlink to the mod, simple as that.<br>

by [Elzzie](https://github.com/lz-fkn), 27.01.2026. MIT License<br>

used stuff:
* [Golang](https://go.dev/) (go1.25.5 windows/amd64)
* [Wails](https://wails.io/) (v2.11.0)
* [Marked.js](https://github.com/markedjs/marked) (v17.0.1)
`;

function renderAbout() {
    const container = document.getElementById('about-content');
    if (container) {
        container.innerHTML = marked.parse(aboutMarkdown);

        container.querySelectorAll('a').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const url = link.getAttribute('href');
                if (url) {
                    window.go.main.App.OpenBrowser(url);
                }
            });
        });
    }
}

window.onload = () => {
    loadMods();
    renderAbout();
    const l1 = document.getElementById('bg-layer-1');
    if (l1) {
        l1.style.backgroundImage = "url('assets/images/bg1.jpg')";
        l1.classList.add('active');
    }
    setInterval(rotateBackground, changeBgInterval);
};