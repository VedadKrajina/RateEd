// Utility
async function api(method, path, body) {
    const opts = { method, headers: {} };
    if (body) {
        opts.headers['Content-Type'] = 'application/json';
        opts.body = JSON.stringify(body);
    }
    const res = await fetch(path, opts);
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Something went wrong');
    return data;
}

function renderStars(avg, count) {
    let html = '';
    const rounded = Math.round(avg);
    for (let i = 1; i <= 5; i++) {
        html += `<span class="${i <= rounded ? 'stars' : 'stars-muted'}">\u2605</span>`;
    }
    if (count !== undefined) {
        html += `<span class="rating-count">(${count} rating${count !== 1 ? 's' : ''})</span>`;
    }
    return html;
}

function timeAgo(dateStr) {
    const d = new Date(dateStr + 'Z');
    const diff = (Date.now() - d.getTime()) / 1000;
    if (diff < 60) return 'just now';
    if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
    if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
    return Math.floor(diff / 86400) + 'd ago';
}

function debounce(fn, ms) {
    let timer;
    return function(...args) {
        clearTimeout(timer);
        timer = setTimeout(() => fn.apply(this, args), ms);
    };
}

let currentUser = null;

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function defaultAvatar() {
    return 'data:image/svg+xml,' + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect fill="%232563EB" width="100" height="100"/><text x="50" y="55" text-anchor="middle" dominant-baseline="middle" fill="%2322C55E" font-size="40" font-family="sans-serif">?</text></svg>');
}

function updatePointsBadge(points) {
    document.querySelectorAll('#points-badge').forEach(el => {
        el.textContent = (points || 0) + ' pts';
    });
}

function rankBadgeHtml(rank) {
    const cls = 'rank-' + rank.toLowerCase();
    return `<span class="rank-badge ${cls}">${escapeHtml(rank)}</span>`;
}

// ==================== LOGIN PAGE ====================
function initLoginPage() {
    const loginTab = document.getElementById('login-tab');
    const registerTab = document.getElementById('register-tab');
    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form');
    const errorEl = document.getElementById('auth-error');

    if (!loginTab) return;

    loginTab.addEventListener('click', () => {
        loginTab.classList.add('active');
        registerTab.classList.remove('active');
        loginForm.style.display = 'block';
        registerForm.style.display = 'none';
        errorEl.textContent = '';
    });

    registerTab.addEventListener('click', () => {
        registerTab.classList.add('active');
        loginTab.classList.remove('active');
        registerForm.style.display = 'block';
        loginForm.style.display = 'none';
        errorEl.textContent = '';
    });

    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        errorEl.textContent = '';
        try {
            await api('POST', '/api/login', {
                username: document.getElementById('login-user').value,
                password: document.getElementById('login-pass').value
            });
            window.location.href = '/home';
        } catch (err) {
            errorEl.textContent = err.message;
        }
    });

    registerForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        errorEl.textContent = '';
        const pass = document.getElementById('reg-pass').value;
        const confirm = document.getElementById('reg-confirm').value;
        if (pass !== confirm) {
            errorEl.textContent = 'Passwords do not match';
            return;
        }
        try {
            await api('POST', '/api/register', {
                username: document.getElementById('reg-user').value,
                password: pass
            });
            window.location.href = '/home';
        } catch (err) {
            errorEl.textContent = err.message;
        }
    });

    api('GET', '/api/me').then(() => {
        window.location.href = '/home';
    }).catch(() => {});
}

// ==================== HOME PAGE ====================
function initHomePage() {
    const instList = document.getElementById('institutions-list');
    const searchInput = document.getElementById('search-input');
    const searchBtn = document.getElementById('search-btn');
    const createInput = document.getElementById('create-input');
    const createBtn = document.getElementById('create-btn');
    const createError = document.getElementById('create-error');
    const usernameEl = document.getElementById('username');
    const logoutBtn = document.getElementById('logout-btn');

    if (!instList) return;

    api('GET', '/api/me').then(user => {
        currentUser = user;
        usernameEl.textContent = user.username;
        usernameEl.style.cursor = 'pointer';
        usernameEl.addEventListener('click', () => {
            window.location.href = '/profile/' + user.username;
        });
        updatePointsBadge(user.contribution_points);

        // Show/hide create section based on points
        const createSection = document.getElementById('create-section');
        const createLocked = document.getElementById('create-locked');
        if (user.contribution_points < 5 && !user.is_admin) {
            createSection.style.display = 'none';
            createLocked.style.display = 'block';
            document.getElementById('points-needed').textContent = 5 - user.contribution_points + ' more';
        } else {
            createSection.style.display = 'block';
            createLocked.style.display = 'none';
        }
    }).catch(() => {
        window.location.href = '/';
    });

    logoutBtn.addEventListener('click', async () => {
        await api('POST', '/api/logout');
        window.location.href = '/';
    });

    async function loadInstitutions(query) {
        const url = query ? `/api/institutions?q=${encodeURIComponent(query)}` : '/api/institutions';
        const items = await api('GET', url);
        if (items.length === 0) {
            instList.innerHTML = '<div class="empty-state">No institutions found. Be the first to add one!</div>';
            return;
        }
        instList.innerHTML = items.map(t => `
            <div class="topic-card" onclick="window.location.href='/institution/${t.id}'">
                ${t.cover_image ? `<img src="/${escapeHtml(t.cover_image)}" class="topic-card-cover" alt="">` : ''}
                <h3>${escapeHtml(t.title)}</h3>
                ${t.institution_type ? `<span class="type-badge">${escapeHtml(t.institution_type)}</span>` : ''}
                <div class="topic-meta">
                    <div>${renderStars(t.avg_rating, t.num_ratings)}</div>
                    <div>by <a href="/profile/${escapeHtml(t.created_by)}" class="username-link" onclick="event.stopPropagation()">${escapeHtml(t.created_by)}</a> &middot; ${timeAgo(t.created_at)}</div>
                </div>
            </div>
        `).join('');
    }

    loadInstitutions('');

    searchBtn.addEventListener('click', () => loadInstitutions(searchInput.value));
    searchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') loadInstitutions(searchInput.value);
    });

    createBtn.addEventListener('click', async () => {
        createError.textContent = '';
        const title = createInput.value.trim();
        if (!title) return;
        const instType = document.getElementById('create-type').value;
        const description = document.getElementById('create-description').value.trim();
        const emailDomain = document.getElementById('create-email-domain').value.trim();
        try {
            const inst = await api('POST', '/api/institutions', { title, institution_type: instType, description, email_domain: emailDomain });
            createInput.value = '';
            document.getElementById('create-description').value = '';
            document.getElementById('create-email-domain').value = '';
            window.location.href = '/institution/' + inst.id;
        } catch (err) {
            createError.textContent = err.message;
        }
    });

    createInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') createBtn.click();
    });

    // School Rankings Sidebar
    async function loadSchoolRankings(type) {
        const container = document.getElementById('school-rankings-list');
        if (!container) return;
        try {
            const url = type ? `/api/schools/rankings?type=${encodeURIComponent(type)}` : '/api/schools/rankings';
            const rankings = await api('GET', url);
            if (rankings.length === 0) {
                container.innerHTML = '<div class="empty-state" style="padding:1rem;font-size:0.9rem">No schools yet</div>';
                return;
            }
            container.innerHTML = rankings.map((s, i) => `
                <div class="school-ranking-entry">
                    <span class="sr-name"><span style="color:#888;margin-right:0.3rem">${i + 1}.</span><a href="/institution/${s.id}">${escapeHtml(s.title)}</a></span>
                    <span class="sr-right">
                        <span class="sr-points">${s.total_points} pts</span>
                    </span>
                </div>
            `).join('');
        } catch {
            container.innerHTML = '<div class="empty-state" style="padding:1rem;font-size:0.9rem">Failed to load</div>';
        }
    }

    loadSchoolRankings('');

    const typeFilter = document.getElementById('school-type-filter');
    if (typeFilter) {
        typeFilter.addEventListener('change', () => {
            loadSchoolRankings(typeFilter.value);
        });
    }
}

// ==================== INSTITUTION PAGE ====================
function initInstitutionPage() {
    const instTitle = document.getElementById('inst-title');
    const instMeta = document.getElementById('inst-meta');
    const instAvg = document.getElementById('inst-avg');
    const instTypeBadge = document.getElementById('inst-type-badge');
    const instDescription = document.getElementById('inst-description');
    const ratingsList = document.getElementById('ratings-list');
    const rateForm = document.getElementById('rate-form');
    const rateComment = document.getElementById('rate-comment');
    const rateError = document.getElementById('rate-error');
    const rateSuccess = document.getElementById('rate-success');

    if (!instTitle) return;

    const pathParts = window.location.pathname.split('/');
    const instId = pathParts[pathParts.length - 1];

    const categoryScores = {};

    document.querySelectorAll('.star-widget[data-category] input').forEach(input => {
        input.addEventListener('change', () => {
            const cat = input.closest('.star-widget').dataset.category;
            categoryScores[cat] = parseInt(input.value);
        });
    });

    const usernameEl = document.getElementById('username');
    const logoutBtn = document.getElementById('logout-btn');
    let currentModeratorId = null;

    async function loadLeaderboard() {
        const container = document.getElementById('school-leaderboard');
        if (!container) return;
        try {
            const data = await api('GET', `/api/institutions/${instId}/leaderboard`);
            const entries = data.entries || [];
            currentModeratorId = data.moderator_id || null;
            if (entries.length === 0) {
                container.innerHTML = '<div class="empty-state" style="padding:1rem;font-size:0.9rem">No contributors yet</div>';
                return;
            }
            const isMod = currentUser && (currentModeratorId === currentUser.id || currentUser.is_admin);
            container.innerHTML = entries.map((e, i) => {
                const modBadge = (e.user_id === currentModeratorId) ? '<span class="moderator-badge">Moderator</span>' : '';
                const muteBtn = (isMod && e.user_id !== currentUser.id)
                    ? `<button class="mute-btn" data-userid="${e.user_id}" data-username="${escapeHtml(e.username)}" title="Mute user">&#128263;</button>`
                    : '';
                return `
                <div class="leaderboard-entry">
                    <span class="lb-name"><span style="color:#888;margin-right:0.3rem">${i + 1}.</span><a href="/profile/${escapeHtml(e.username)}">${escapeHtml(e.username)}</a>${modBadge}</span>
                    <span class="lb-right">
                        ${muteBtn}
                        <span class="lb-points">${e.points} pts</span>
                        ${rankBadgeHtml(e.rank)}
                    </span>
                </div>
            `;
            }).join('');
            bindMuteButtons();
        } catch {
            container.innerHTML = '<div class="empty-state" style="padding:1rem;font-size:0.9rem">Failed to load</div>';
        }
    }

    function bindMuteButtons() {
        document.querySelectorAll('.mute-btn').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                e.stopPropagation();
                const targetUserId = parseInt(btn.dataset.userid);
                const username = btn.dataset.username;
                const minutes = prompt(`Mute ${username} for how many minutes?`, '60');
                if (!minutes) return;
                const duration = parseInt(minutes);
                if (isNaN(duration) || duration < 1) { alert('Invalid duration'); return; }
                try {
                    await api('POST', `/api/institutions/${instId}/mute`, { user_id: targetUserId, duration: duration });
                    alert(`${username} has been muted for ${duration} minutes`);
                } catch (err) {
                    alert(err.message);
                }
            });
        });
    }

    api('GET', '/api/me').then(user => {
        currentUser = user;
        usernameEl.textContent = user.username;
        usernameEl.style.cursor = 'pointer';
        usernameEl.addEventListener('click', () => {
            window.location.href = '/profile/' + user.username;
        });
        updatePointsBadge(user.contribution_points);
        loadInstitution();
        loadDiscussion();
        loadLeaderboard();
    }).catch(() => {
        window.location.href = '/';
    });

    logoutBtn.addEventListener('click', async () => {
        await api('POST', '/api/logout');
        window.location.href = '/';
    });

    async function loadInstitution() {
        try {
            const t = await api('GET', `/api/institutions/${instId}`);
            instTitle.textContent = t.title;

            // Cover image
            const existingCover = document.getElementById('inst-cover-img');
            if (existingCover) existingCover.remove();
            if (t.cover_image) {
                const coverImg = document.createElement('img');
                coverImg.id = 'inst-cover-img';
                coverImg.src = '/' + t.cover_image;
                coverImg.className = 'topic-cover';
                coverImg.alt = '';
                const header = document.querySelector('.institution-header');
                header.parentNode.insertBefore(coverImg, header);
            }

            if (t.institution_type) {
                instTypeBadge.innerHTML = `<span class="type-badge-large">${escapeHtml(t.institution_type)}</span>`;
            }
            if (t.description) {
                instDescription.textContent = t.description;
            }

            instMeta.innerHTML = `Added by <a href="/profile/${escapeHtml(t.created_by)}" class="username-link">${escapeHtml(t.created_by)}</a> &middot; ${timeAgo(t.created_at)}`;
            instAvg.innerHTML = renderStars(t.avg_rating, t.num_ratings) +
                (t.avg_rating > 0 ? ` <strong>${t.avg_rating.toFixed(1)}</strong>` : '');

            // Cover upload
            let coverSection = document.getElementById('cover-upload-section');
            const canUploadCover = currentUser && (currentUser.contribution_points >= 5 || currentModeratorId === currentUser.id || currentUser.is_admin);
            if (!coverSection && canUploadCover) {
                coverSection = document.createElement('div');
                coverSection.id = 'cover-upload-section';
                coverSection.className = 'cover-upload-section';
                coverSection.innerHTML = `
                    <label for="cover-input" class="btn btn-accent btn-small" style="cursor:pointer">Upload Cover Image</label>
                    <input type="file" id="cover-input" accept="image/*" style="display:none">
                `;
                document.querySelector('.institution-header').appendChild(coverSection);
                document.getElementById('cover-input').addEventListener('change', async (e) => {
                    const file = e.target.files[0];
                    if (!file) return;
                    const formData = new FormData();
                    formData.append('cover', file);
                    try {
                        const res = await fetch(`/api/institutions/${instId}/cover`, { method: 'POST', body: formData });
                        const data = await res.json();
                        if (!res.ok) throw new Error(data.error);
                        loadInstitution();
                    } catch (err) {
                        alert(err.message);
                    }
                });
            }

            // Verification section
            const verifySection = document.getElementById('verify-section');
            if (t.email_domain && currentUser) {
                // Check if already verified
                const alreadyVerified = t.ratings.some(r => r.user_id === currentUser.id && r.is_verified);
                if (alreadyVerified) {
                    verifySection.style.display = 'block';
                    verifySection.innerHTML = '<h3>Verification</h3><p style="color:#22C55E;font-weight:600">&#10003; You are verified for this institution</p>';
                } else {
                    verifySection.style.display = 'block';
                    document.getElementById('verify-email').placeholder = `your.name${t.email_domain}`;
                }
            }

            if (t.ratings.length === 0) {
                ratingsList.innerHTML = '<div class="empty-state">No ratings yet. Be the first!</div>';
            } else {
                ratingsList.innerHTML = t.ratings.map(r => {
                    const categoryLabels = {
                        academic: 'Academic Rigor & Quality',
                        infrastructure: 'Infrastructure & Resources',
                        student_life: 'Student Life & Environment',
                        career: 'Career & Future Support',
                        guidance: 'Academic Guidance & Staff'
                    };
                    const catScores = {
                        academic: r.score_academic,
                        infrastructure: r.score_infrastructure,
                        student_life: r.score_student_life,
                        career: r.score_career,
                        guidance: r.score_guidance
                    };
                    const hasBreakdown = Object.values(catScores).some(v => v > 0);
                    let breakdownHtml = '';
                    if (hasBreakdown) {
                        breakdownHtml = `
                            <button class="breakdown-toggle-btn" onclick="this.nextElementSibling.style.display = this.nextElementSibling.style.display === 'none' ? 'block' : 'none'">See breakdown</button>
                            <div class="rating-breakdown" style="display:none">
                                ${Object.entries(catScores).map(([key, val]) => `
                                    <div class="breakdown-row">
                                        <span class="breakdown-label">${categoryLabels[key]}</span>
                                        <span class="breakdown-stars"><span class="stars">${'\u2605'.repeat(val)}</span><span class="stars-muted">${'\u2605'.repeat(5 - val)}</span></span>
                                    </div>
                                `).join('')}
                            </div>
                        `;
                    }
                    const verifiedBadge = r.is_verified ? '<span class="verified-badge" title="Verified">&#10003; Verified</span>' : '';
                    return `
                        <div class="comment-card">
                            <div class="comment-header">
                                <span>
                                    <a href="/profile/${escapeHtml(r.username)}" class="username-link">${escapeHtml(r.username)}</a>
                                    ${verifiedBadge}
                                    <span class="breakdown-stars" style="margin-left:0.5rem"><span class="stars">${'\u2605'.repeat(r.score)}</span><span class="stars-muted">${'\u2605'.repeat(5 - r.score)}</span></span>
                                </span>
                                <span class="comment-date">${timeAgo(r.created_at)}</span>
                            </div>
                            ${r.comment ? `<div class="comment-text">${escapeHtml(r.comment)}</div>` : ''}
                            ${breakdownHtml}
                        </div>
                    `;
                }).join('');
            }
        } catch {
            instTitle.textContent = 'Institution not found';
        }
    }

    rateForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        rateError.textContent = '';
        rateSuccess.textContent = '';

        const requiredCats = ['academic', 'infrastructure', 'student_life', 'career', 'guidance'];
        const missing = requiredCats.filter(c => !categoryScores[c]);
        if (missing.length > 0) {
            rateError.textContent = 'Please rate all 5 categories';
            return;
        }

        try {
            await api('POST', `/api/institutions/${instId}/rate`, {
                score_academic: categoryScores.academic,
                score_infrastructure: categoryScores.infrastructure,
                score_student_life: categoryScores.student_life,
                score_career: categoryScores.career,
                score_guidance: categoryScores.guidance,
                comment: rateComment.value.trim()
            });
            rateSuccess.textContent = 'Rating submitted!';
            rateComment.value = '';
            requiredCats.forEach(c => delete categoryScores[c]);
            document.querySelectorAll('.star-widget[data-category] input').forEach(i => i.checked = false);
            // Refresh points
            const me = await api('GET', '/api/me');
            currentUser = me;
            updatePointsBadge(me.contribution_points);
            loadInstitution();
            loadLeaderboard();
        } catch (err) {
            rateError.textContent = err.message;
        }
    });

    // ==================== Verification ====================
    const verifyBtn = document.getElementById('verify-btn');
    if (verifyBtn) {
        verifyBtn.addEventListener('click', async () => {
            const verifyError = document.getElementById('verify-error');
            const verifySuccess = document.getElementById('verify-success');
            verifyError.textContent = '';
            verifySuccess.textContent = '';
            const email = document.getElementById('verify-email').value.trim();
            if (!email) {
                verifyError.textContent = 'Please enter your email';
                return;
            }
            try {
                await api('POST', `/api/institutions/${instId}/verify`, { email });
                verifySuccess.textContent = 'Verification successful!';
                loadInstitution();
            } catch (err) {
                verifyError.textContent = err.message;
            }
        });
    }

    // ==================== Discussion ====================
    async function loadDiscussion() {
        const discussionList = document.getElementById('discussion-list');
        try {
            const comments = await api('GET', `/api/institutions/${instId}/discussion`);
            if (comments.length === 0) {
                discussionList.innerHTML = '<div class="empty-state">No discussion yet. Start the conversation!</div>';
                return;
            }
            discussionList.innerHTML = renderCommentTree(comments);
            bindReplyButtons();
        } catch {
            discussionList.innerHTML = '<div class="empty-state">Failed to load discussion.</div>';
        }
    }

    function renderCommentTree(comments) {
        // Build tree
        const byId = {};
        const roots = [];
        comments.forEach(c => { byId[c.id] = { ...c, children: [] }; });
        comments.forEach(c => {
            if (c.parent_id && byId[c.parent_id]) {
                byId[c.parent_id].children.push(byId[c.id]);
            } else {
                roots.push(byId[c.id]);
            }
        });

        const isMod = currentUser && (currentModeratorId === currentUser.id || currentUser.is_admin);

        function renderNode(node, isReply) {
            const cls = isReply ? 'discussion-comment discussion-reply' : 'discussion-comment';
            const isOwn = currentUser && node.user_id === currentUser.id;
            const modBadge = (node.user_id === currentModeratorId) ? '<span class="moderator-badge">Moderator</span>' : '';
            const pointsBadge = `<span class="comment-points">${node.contribution_points} pts</span>`;
            const deleteBtn = isOwn ? `<button class="comment-delete-btn" data-id="${node.id}" title="Delete your comment">&times;</button>` : '';
            const modDeleteBtn = (isMod && !isOwn) ? `<button class="comment-delete-btn mod-delete" data-id="${node.id}" title="Delete as moderator">&times;</button>` : '';
            let html = `
                <div class="${cls}" data-id="${node.id}">
                    <div class="comment-header">
                        <span>
                            <a href="/profile/${escapeHtml(node.username)}" class="username-link">${escapeHtml(node.username)}</a>
                            ${modBadge}
                            ${pointsBadge}
                        </span>
                        <span class="comment-header-right">
                            <span class="comment-date">${timeAgo(node.created_at)}</span>
                            ${deleteBtn}
                            ${modDeleteBtn}
                        </span>
                    </div>
                    <div class="comment-text">${escapeHtml(node.content)}</div>
                    <button class="reply-btn" data-id="${node.id}">Reply</button>
                    <div class="inline-reply-form" id="reply-form-${node.id}">
                        <textarea placeholder="Write a reply..." id="reply-text-${node.id}"></textarea>
                        <button class="btn btn-accent btn-small submit-reply-btn" data-id="${node.id}">Post Reply</button>
                    </div>
            `;
            if (node.children.length > 0) {
                html += node.children.map(c => renderNode(c, true)).join('');
            }
            html += '</div>';
            return html;
        }

        return roots.map(r => renderNode(r, false)).join('');
    }

    function bindReplyButtons() {
        document.querySelectorAll('.reply-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const form = document.getElementById(`reply-form-${btn.dataset.id}`);
                form.style.display = form.style.display === 'none' || form.style.display === '' ? 'block' : 'none';
            });
        });

        document.querySelectorAll('.submit-reply-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const parentId = parseInt(btn.dataset.id);
                const text = document.getElementById(`reply-text-${parentId}`).value.trim();
                if (!text) return;
                try {
                    await api('POST', `/api/institutions/${instId}/discussion`, {
                        content: text,
                        parent_id: parentId
                    });
                    const me = await api('GET', '/api/me');
                    currentUser = me;
                    updatePointsBadge(me.contribution_points);
                    loadDiscussion();
                    loadLeaderboard();
                } catch (err) {
                    alert(err.message);
                }
            });
        });

        // Own comment delete buttons
        document.querySelectorAll('.comment-delete-btn:not(.mod-delete)').forEach(btn => {
            btn.addEventListener('click', async () => {
                if (!confirm('Delete this comment? Your points from it will be removed.')) return;
                try {
                    await api('DELETE', `/api/institutions/${instId}/discussion/${btn.dataset.id}`);
                    const me = await api('GET', '/api/me');
                    currentUser = me;
                    updatePointsBadge(me.contribution_points);
                    loadDiscussion();
                    loadLeaderboard();
                } catch (err) {
                    alert(err.message);
                }
            });
        });

        // Mod delete buttons
        document.querySelectorAll('.comment-delete-btn.mod-delete').forEach(btn => {
            btn.addEventListener('click', async () => {
                if (!confirm('Delete this comment as moderator?')) return;
                try {
                    await api('DELETE', `/api/institutions/${instId}/discussion/${btn.dataset.id}/mod`);
                    const me = await api('GET', '/api/me');
                    currentUser = me;
                    updatePointsBadge(me.contribution_points);
                    loadDiscussion();
                    loadLeaderboard();
                } catch (err) {
                    alert(err.message);
                }
            });
        });
    }

    // Post top-level comment
    const postBtn = document.getElementById('discussion-post-btn');
    const discussionInput = document.getElementById('discussion-input');
    if (postBtn) {
        postBtn.addEventListener('click', async () => {
            const content = discussionInput.value.trim();
            if (!content) return;
            try {
                await api('POST', `/api/institutions/${instId}/discussion`, {
                    content: content,
                    parent_id: null
                });
                discussionInput.value = '';
                // Remove muted notice if present
                const notice = document.getElementById('muted-notice');
                if (notice) notice.remove();
                const me = await api('GET', '/api/me');
                currentUser = me;
                updatePointsBadge(me.contribution_points);
                loadDiscussion();
                loadLeaderboard();
            } catch (err) {
                // Show muted notice if it's a mute error
                if (err.message && err.message.startsWith('You are muted')) {
                    let notice = document.getElementById('muted-notice');
                    if (!notice) {
                        notice = document.createElement('div');
                        notice.id = 'muted-notice';
                        notice.className = 'muted-notice';
                        const form = document.querySelector('.discussion-post-form');
                        if (form) form.parentNode.insertBefore(notice, form);
                    }
                    notice.textContent = err.message;
                } else {
                    alert(err.message);
                }
            }
        });
    }
}

// ==================== PROFILE PAGE ====================
function initProfilePage() {
    const profileUsername = document.getElementById('profile-username');
    if (!profileUsername) return;

    const pathParts = window.location.pathname.split('/');
    const profileName = decodeURIComponent(pathParts[pathParts.length - 1]);

    const usernameEl = document.getElementById('username');
    const logoutBtn = document.getElementById('logout-btn');

    api('GET', '/api/me').then(user => {
        currentUser = user;
        usernameEl.textContent = user.username;
        usernameEl.style.cursor = 'pointer';
        usernameEl.addEventListener('click', () => {
            window.location.href = '/profile/' + user.username;
        });
        updatePointsBadge(user.contribution_points);
        loadProfile();
    }).catch(() => {
        window.location.href = '/';
    });

    logoutBtn.addEventListener('click', async () => {
        await api('POST', '/api/logout');
        window.location.href = '/';
    });

    async function loadProfile() {
        try {
            const p = await api('GET', `/api/users/${encodeURIComponent(profileName)}`);
            const isOwnProfile = currentUser && currentUser.id === p.user_id;

            profileUsername.textContent = p.username;
            document.getElementById('profile-points').textContent = p.contribution_points + ' pts';
            document.getElementById('profile-rating-count').textContent = p.rating_count + ' rating' + (p.rating_count !== 1 ? 's' : '');

            // Profile picture
            const picEl = document.getElementById('profile-picture');
            picEl.src = p.profile_picture ? '/' + p.profile_picture : defaultAvatar();

            if (isOwnProfile) {
                document.getElementById('picture-upload-controls').style.display = 'flex';
                document.getElementById('picture-input').addEventListener('change', async (e) => {
                    const file = e.target.files[0];
                    if (!file) return;
                    const formData = new FormData();
                    formData.append('picture', file);
                    try {
                        const res = await fetch('/api/profile/picture', { method: 'POST', body: formData });
                        const data = await res.json();
                        if (!res.ok) throw new Error(data.error);
                        picEl.src = data.path + '?t=' + Date.now();
                    } catch (err) {
                        alert(err.message);
                    }
                });
                document.getElementById('remove-picture-btn').addEventListener('click', async () => {
                    try {
                        await api('DELETE', '/api/profile/picture');
                        picEl.src = defaultAvatar();
                    } catch (err) {
                        alert(err.message);
                    }
                });
            }

            // Institution affiliations
            const instList = document.getElementById('profile-institutions-list');
            if (p.institutions && p.institutions.length > 0) {
                instList.innerHTML = p.institutions.map(inst => `
                    <div class="affiliation-item" onclick="window.location.href='/institution/${inst.id}'">
                        <span>
                            <span class="name">${escapeHtml(inst.title)}</span>
                            ${inst.institution_type ? `<span class="type-badge">${escapeHtml(inst.institution_type)}</span>` : ''}
                        </span>
                        <span>${renderStars(inst.avg_rating)}</span>
                    </div>
                `).join('');
            } else {
                instList.innerHTML = '<div class="empty-state" style="padding:1rem">No institution affiliations yet</div>';
            }

            // Education history
            loadEducation(p.education || [], isOwnProfile);

            // Admin panel
            if (currentUser && currentUser.is_admin) {
                initAdminPanel();
            }

        } catch {
            profileUsername.textContent = 'User not found';
        }
    }

    function loadEducation(entries, isOwnProfile) {
        const list = document.getElementById('education-list');
        const formContainer = document.getElementById('edu-form-container');

        if (isOwnProfile && formContainer) {
            formContainer.style.display = 'block';
            initEducationForm();
        }

        if (entries.length === 0) {
            list.innerHTML = '<div class="empty-state" style="padding:1rem">No education history yet</div>';
            return;
        }

        list.innerHTML = entries.map(e => {
            const instLink = e.institution_id
                ? `<a href="/institution/${e.institution_id}" class="username-link edu-inst-name">${escapeHtml(e.institution_name)}</a>`
                : `<span class="edu-inst-name">${escapeHtml(e.institution_name)}</span>`;
            const dateRange = e.end_date ? `${e.start_date} - ${e.end_date}` : `${e.start_date} - Present`;
            const deleteBtn = isOwnProfile
                ? `<button class="edu-delete-btn" data-id="${e.id}" title="Remove">&times;</button>`
                : '';
            return `
                <div class="edu-entry">
                    <div class="edu-entry-main">
                        <span class="edu-role-badge">${escapeHtml(e.role)}</span>
                        ${instLink}
                        <span class="edu-dates">${escapeHtml(dateRange)}</span>
                    </div>
                    ${deleteBtn}
                </div>
            `;
        }).join('');

        if (isOwnProfile) {
            list.querySelectorAll('.edu-delete-btn').forEach(btn => {
                btn.addEventListener('click', async () => {
                    try {
                        await api('DELETE', `/api/profile/education/${btn.dataset.id}`);
                        // Reload profile
                        const p = await api('GET', `/api/users/${encodeURIComponent(profileName)}`);
                        loadEducation(p.education || [], true);
                    } catch (err) {
                        alert(err.message);
                    }
                });
            });
        }
    }

    function initEducationForm() {
        const nameInput = document.getElementById('edu-inst-name');
        const roleSelect = document.getElementById('edu-role');
        const startInput = document.getElementById('edu-start-date');
        const endInput = document.getElementById('edu-end-date');
        const submitBtn = document.getElementById('edu-submit-btn');
        const errorEl = document.getElementById('edu-error');
        const autocomplete = document.getElementById('edu-autocomplete');

        if (!nameInput || nameInput.dataset.bound) return;
        nameInput.dataset.bound = '1';

        // Autocomplete
        const doSearch = debounce(async () => {
            const q = nameInput.value.trim();
            if (q.length < 2) { autocomplete.innerHTML = ''; return; }
            try {
                const results = await api('GET', `/api/institutions?q=${encodeURIComponent(q)}`);
                if (results.length === 0) { autocomplete.innerHTML = ''; return; }
                autocomplete.innerHTML = results.slice(0, 5).map(r =>
                    `<div class="edu-autocomplete-item" data-name="${escapeHtml(r.title)}">${escapeHtml(r.title)}</div>`
                ).join('');
                autocomplete.querySelectorAll('.edu-autocomplete-item').forEach(item => {
                    item.addEventListener('mousedown', (e) => {
                        e.preventDefault();
                        nameInput.value = item.dataset.name;
                        autocomplete.innerHTML = '';
                    });
                });
            } catch { autocomplete.innerHTML = ''; }
        }, 250);

        nameInput.addEventListener('input', doSearch);
        nameInput.addEventListener('blur', () => {
            setTimeout(() => { autocomplete.innerHTML = ''; }, 150);
        });

        submitBtn.addEventListener('click', async () => {
            errorEl.textContent = '';
            const instName = nameInput.value.trim();
            const role = roleSelect.value;
            const startDate = startInput.value;
            const endDate = endInput.value;

            if (!instName || !startDate || !role) {
                errorEl.textContent = 'Institution name, start date, and role are required';
                return;
            }

            try {
                await api('POST', '/api/profile/education', {
                    institution_name: instName,
                    start_date: startDate,
                    end_date: endDate,
                    role: role
                });
                nameInput.value = '';
                roleSelect.value = '';
                startInput.value = '';
                endInput.value = '';
                // Reload
                const p = await api('GET', `/api/users/${encodeURIComponent(profileName)}`);
                loadEducation(p.education || [], true);
            } catch (err) {
                errorEl.textContent = err.message;
            }
        });
    }
}

// ==================== ADMIN PANEL ====================
async function initAdminPanel() {
    const panel = document.getElementById('admin-panel');
    if (!panel) return;
    panel.style.display = 'block';

    try {
        const users = await api('GET', '/api/admin/users');
        const list = document.getElementById('admin-users-list');
        list.innerHTML = users.map(u => `
            <div class="admin-user-row">
                <div class="admin-user-info">
                    <a href="/profile/${escapeHtml(u.username)}" class="user-name username-link">${escapeHtml(u.username)}</a>
                    <span style="color:#888;font-size:0.85rem">${u.rating_count} ratings</span>
                </div>
                <div class="admin-score-control">
                    <span style="font-size:0.85rem;color:#888">pts:</span>
                    <input type="number" value="${u.contribution_points}" data-userid="${u.id}">
                    <button class="btn btn-accent btn-small admin-set-pts-btn" data-userid="${u.id}">Set</button>
                </div>
            </div>
        `).join('');

        list.querySelectorAll('.admin-set-pts-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const input = btn.parentElement.querySelector('input[type="number"]');
                try {
                    await api('PUT', `/api/admin/users/${btn.dataset.userid}/points`, {
                        points: parseInt(input.value)
                    });
                    btn.textContent = 'Done!';
                    setTimeout(() => { btn.textContent = 'Set'; }, 1000);
                } catch (err) {
                    alert(err.message);
                }
            });
        });
    } catch (err) {
        console.error('Failed to load admin panel:', err);
    }
}

// Init on page load
document.addEventListener('DOMContentLoaded', () => {
    initLoginPage();
    initHomePage();
    initInstitutionPage();
    initProfilePage();
});
