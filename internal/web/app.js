// Task Manager Web UI
(function() {
    'use strict';

    let allTasks = [];
    let filteredTasks = [];
    let tags = new Set();

    // DOM Elements
    const taskListEl = document.getElementById('task-list');
    const statusFilter = document.getElementById('status-filter');
    const tagFilter = document.getElementById('tag-filter');
    const searchInput = document.getElementById('search-input');
    const sortSelect = document.getElementById('sort-select');
    const refreshBtn = document.getElementById('refresh-btn');
    const themeToggle = document.getElementById('theme-toggle');
    const modal = document.getElementById('task-modal');
    const modalClose = document.getElementById('modal-close');
    const modalTitle = document.getElementById('modal-title');
    const modalStatus = document.getElementById('modal-status');
    const modalCreated = document.getElementById('modal-created');
    const modalTags = document.getElementById('modal-tags');
    const modalTagsItem = document.getElementById('modal-tags-item');
    const modalDeps = document.getElementById('modal-deps');
    const modalDepsItem = document.getElementById('modal-deps-item');
    const modalDetail = document.getElementById('modal-detail');

    // Initialize
    async function init() {
        loadTheme();
        await loadTasks();
        setupEventListeners();
    }

    // Load tasks from API
    async function loadTasks() {
        try {
            const response = await fetch('/api/tasks');
            if (!response.ok) {
                throw new Error('Failed to load tasks');
            }
            allTasks = await response.json();
            extractTags();
            populateTagFilter();
            applyFiltersAndSort();
            updateStats();
        } catch (error) {
            console.error('Error loading tasks:', error);
            taskListEl.innerHTML = `<div class="empty-state"><h3>Error loading tasks</h3><p>${escapeHtml(error.message)}</p></div>`;
        }
    }

    // Extract all unique tags
    function extractTags() {
        tags = new Set();
        allTasks.forEach(task => {
            if (task.tags) {
                task.tags.forEach(tag => tags.add(tag));
            }
        });
    }

    // Populate tag filter dropdown
    function populateTagFilter() {
        const currentValue = tagFilter.value;
        tagFilter.innerHTML = '<option value="">All</option>';
        Array.from(tags).sort().forEach(tag => {
            const option = document.createElement('option');
            option.value = tag;
            option.textContent = tag;
            tagFilter.appendChild(option);
        });
        tagFilter.value = currentValue;
    }

    // Apply filters and sorting
    function applyFiltersAndSort() {
        filteredTasks = [...allTasks];

        // Filter by status
        const statusValue = statusFilter.value;
        if (statusValue) {
            filteredTasks = filteredTasks.filter(t => t.status === statusValue);
        }

        // Filter by tag
        const tagValue = tagFilter.value;
        if (tagValue) {
            filteredTasks = filteredTasks.filter(t => t.tags && t.tags.includes(tagValue));
        }

        // Filter by search
        const searchTerm = searchInput.value.toLowerCase().trim();
        if (searchTerm) {
            filteredTasks = filteredTasks.filter(t =>
                t.title.toLowerCase().includes(searchTerm) ||
                t.id.toLowerCase().includes(searchTerm)
            );
        }

        // Sort
        const sortValue = sortSelect.value;
        filteredTasks.sort((a, b) => {
            switch (sortValue) {
                case 'id-asc':
                    return (parseInt(a.id.replace(/\D/g, '')) || 0) - (parseInt(b.id.replace(/\D/g, '')) || 0);
                case 'id-desc':
                    return (parseInt(b.id.replace(/\D/g, '')) || 0) - (parseInt(a.id.replace(/\D/g, '')) || 0);
                case 'created-asc':
                    return new Date(a.created_at) - new Date(b.created_at);
                case 'created-desc':
                    return new Date(b.created_at) - new Date(a.created_at);
                case 'title-asc':
                    return a.title.localeCompare(b.title);
                case 'title-desc':
                    return b.title.localeCompare(a.title);
                default:
                    return 0;
            }
        });

        renderTasks();
    }

    // Render tasks
    function renderTasks() {
        if (filteredTasks.length === 0) {
            taskListEl.innerHTML = '<div class="empty-state"><h3>No tasks found</h3><p>Try adjusting your filters or add tasks with `tssk add`</p></div>';
            return;
        }

        taskListEl.innerHTML = filteredTasks.map(task => createTaskCard(task)).join('');

        // Add click handlers
        taskListEl.querySelectorAll('.task-card').forEach(card => {
            card.addEventListener('click', () => openTask(card.dataset.taskId));
        });
    }

    // Create task card HTML
    function createTaskCard(task) {
        const statusClass = `status-${task.status}`;
        const statusLabel = task.status.replace('-', ' ');
        const tagsHtml = (task.tags && task.tags.length > 0)
            ? `<div class="task-tags">${task.tags.map(t => `<span class="tag">${escapeHtml(t)}</span>`).join('')}</div>`
            : '';
        const depsHtml = (task.dependencies && task.dependencies.length > 0)
            ? `<div class="task-deps">Depends on: ${task.dependencies.map(d => escapeHtml(d)).join(', ')}</div>`
            : '';

        return `
            <div class="task-card ${statusClass}" data-task-id="${task.id}">
                <div class="task-header">
                    <span class="task-id">${task.id}</span>
                    <span class="task-title">${escapeHtml(task.title)}</span>
                    <span class="task-status ${statusClass}">${statusLabel}</span>
                </div>
                <div class="task-meta-row">
                    ${tagsHtml}
                    ${depsHtml}
                </div>
            </div>
        `;
    }

    // Update statistics
    function updateStats() {
        document.getElementById('stat-total').textContent = allTasks.length;
        document.getElementById('stat-todo').textContent = allTasks.filter(t => t.status === 'todo').length;
        document.getElementById('stat-in-progress').textContent = allTasks.filter(t => t.status === 'in-progress').length;
        document.getElementById('stat-done').textContent = allTasks.filter(t => t.status === 'done').length;
        document.getElementById('stat-blocked').textContent = allTasks.filter(t => t.status === 'blocked').length;
    }

    // Open task detail modal
    async function openTask(taskId) {
        try {
            const response = await fetch(`/api/tasks/${taskId}`);
            if (!response.ok) {
                throw new Error('Failed to load task');
            }
            const task = await response.json();

            modalTitle.textContent = `${task.id}: ${task.title}`;

            // Status dropdown
            modalStatus.innerHTML = [
                { value: 'todo', label: 'Todo' },
                { value: 'in-progress', label: 'In Progress' },
                { value: 'done', label: 'Done' },
                { value: 'blocked', label: 'Blocked' }
            ].map(s => `<option value="${s.value}" ${task.status === s.value ? 'selected' : ''}>${s.label}</option>`).join('');

            // Created date
            const createdDate = new Date(task.created_at);
            modalCreated.textContent = createdDate.toLocaleString();

            // Tags
            if (task.tags && task.tags.length > 0) {
                modalTagsItem.style.display = 'flex';
                modalTags.innerHTML = task.tags.map(t => `<span class="tag">${escapeHtml(t)}</span>`).join('');
            } else {
                modalTagsItem.style.display = 'none';
            }

            // Dependencies
            if (task.dependencies && task.dependencies.length > 0) {
                modalDepsItem.style.display = 'flex';
                modalDeps.innerHTML = task.dependencies.map(d => `<span class="tag">${escapeHtml(d)}</span>`).join('');
            } else {
                modalDepsItem.style.display = 'none';
            }

            // Detail markdown
            if (task.detail) {
                modalDetail.innerHTML = sanitizeHtml(marked.parse(task.detail));
            } else {
                modalDetail.innerHTML = '<p style="color: var(--text-secondary);">No detail text available.</p>';
            }

            // Status change handler
            modalStatus.onchange = async () => {
                try {
                    const newStatus = modalStatus.value;
                    const updateResponse = await fetch(`/api/tasks/${task.id}/status`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ status: newStatus })
                    });
                    if (!updateResponse.ok) {
                        throw new Error('Failed to update status');
                    }
                    await loadTasks();
                    modalTitle.textContent = `${task.id}: ${task.title}`;
                } catch (error) {
                    console.error('Error updating status:', error);
                    alert('Failed to update task status');
                }
            };

            modal.classList.add('active');
        } catch (error) {
            console.error('Error loading task:', error);
            alert('Failed to load task details');
        }
    }

    // Close modal
    function closeModal() {
        modal.classList.remove('active');
    }

    // Theme management
    function loadTheme() {
        const savedTheme = localStorage.getItem('tssk-theme') || 'light';
        document.documentElement.setAttribute('data-theme', savedTheme);
        themeToggle.textContent = savedTheme === 'dark' ? '☀️' : '🌙';
    }

    function toggleTheme() {
        const current = document.documentElement.getAttribute('data-theme');
        const newTheme = current === 'dark' ? 'light' : 'dark';
        document.documentElement.setAttribute('data-theme', newTheme);
        localStorage.setItem('tssk-theme', newTheme);
        themeToggle.textContent = newTheme === 'dark' ? '☀️' : '🌙';
    }

    // Sanitize HTML to prevent XSS from rendered markdown
    function sanitizeHtml(html) {
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, 'text/html');
        const dangerous = doc.querySelectorAll('script,style,link,object,embed,iframe,form');
        dangerous.forEach(el => el.remove());
        doc.querySelectorAll('*').forEach(el => {
            Array.from(el.attributes).forEach(attr => {
                if (attr.name.startsWith('on') ||
                    (attr.name === 'href' && /^javascript:/i.test(attr.value)) ||
                    (attr.name === 'src' && /^javascript:/i.test(attr.value))) {
                    el.removeAttribute(attr.name);
                }
            });
        });
        return doc.body.innerHTML;
    }

    // Escape HTML
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Event listeners
    function setupEventListeners() {
        statusFilter.addEventListener('change', applyFiltersAndSort);
        tagFilter.addEventListener('change', applyFiltersAndSort);
        searchInput.addEventListener('input', applyFiltersAndSort);
        sortSelect.addEventListener('change', applyFiltersAndSort);
        refreshBtn.addEventListener('click', loadTasks);
        themeToggle.addEventListener('click', toggleTheme);
        modalClose.addEventListener('click', closeModal);
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                closeModal();
            }
        });
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && modal.classList.contains('active')) {
                closeModal();
            }
        });
    }

    // Start the app
    init();
})();
