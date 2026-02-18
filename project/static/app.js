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
let currentLang = localStorage.getItem('lang') || 'en';
let currentCurrency = 'USD';

const CURRENCY_RATES = { USD: 1, KM: 1.77, EUR: 0.92 };
const CURRENCY_SYMBOLS = { USD: '$', KM: 'KM', EUR: '€' };

function formatTuition(usdMin, usdMax, currency) {
    const rate = CURRENCY_RATES[currency];
    const sym = CURRENCY_SYMBOLS[currency];
    const fmt = v => currency === 'KM'
        ? `${Math.round(v * rate).toLocaleString()} KM`
        : `${sym}${Math.round(v * rate).toLocaleString()}`;
    if (usdMin && usdMax) return `${fmt(usdMin)}–${fmt(usdMax)}/yr`;
    if (usdMin) return `From ${fmt(usdMin)}/yr`;
    if (usdMax) return `Up to ${fmt(usdMax)}/yr`;
    return '';
}

const TRANSLATIONS = {
    en: {
        logout: 'Logout', login: 'Login', cancel: 'Cancel',
        loginRequired: 'Login Required',
        loginToRate: 'You need to be logged in to rate institutions.',
        verifyToRate: 'You must be verified for this institution to leave a rating. Use the verification section above to get verified.',
        rateTitle: 'Rate This Institution',
        submitRating: 'Submit Rating',
        ratingsTitle: 'Ratings',
        discussionTitle: 'Discussion',
        joinDiscussion: 'Join the discussion...',
        postComment: 'Post Comment',
        verified: '\u2713 Verified',
        noRatings: 'No ratings yet. Be the first!',
        leaderboard: 'Leaderboard',
        backToInstitutions: '\u2190 Back to institutions',
        programs: 'Programs Offered',
        pros: '\u2713 Pros', cons: '\u2717 Cons',
        verifyAffiliation: 'Verify Your Affiliation',
        emailVerification: 'Email Verification',
        uploadPhotoProof: 'Upload Photo Proof',
        editDetails: 'Edit Details',
        save: 'Save',
        addPhoto: 'Add Photo',
        pendingVerifications: 'Pending Verification Requests',
        searchPlaceholder: 'Search institutions...',
        addInstitution: 'Add Institution',
        createInstitution: 'Add New Institution',
    },
    bs: {
        logout: 'Odjava', login: 'Prijava', cancel: 'Odustani',
        loginRequired: 'Prijavite se',
        loginToRate: 'Morate biti prijavljeni da biste ocijenili instituciju.',
        verifyToRate: 'Morate biti verifikovani za ovu instituciju da biste ostavili ocjenu. Koristite gore navedenu sekciju za verifikaciju.',
        rateTitle: 'Ocijenite instituciju',
        submitRating: 'Pošalji ocjenu',
        ratingsTitle: 'Ocjene',
        discussionTitle: 'Diskusija',
        joinDiscussion: 'Pridružite se diskusiji...',
        postComment: 'Objavi komentar',
        verified: '\u2713 Verifikovan',
        noRatings: 'Još nema ocjena. Budite prvi!',
        leaderboard: 'Rang lista',
        backToInstitutions: '\u2190 Nazad na institucije',
        programs: 'Programi',
        pros: '\u2713 Prednosti', cons: '\u2717 Nedostaci',
        verifyAffiliation: 'Verifikujte svoju povezanost',
        emailVerification: 'Email verifikacija',
        uploadPhotoProof: 'Otpremite foto dokaz',
        editDetails: 'Uredi detalje',
        save: 'Sačuvaj',
        addPhoto: 'Dodaj fotografiju',
        pendingVerifications: 'Zahtjevi za verifikaciju',
        searchPlaceholder: 'Pretraži institucije...',
        addInstitution: 'Dodaj instituciju',
        createInstitution: 'Dodaj novu instituciju',
    }
};

function applyTranslations() {
    const t = TRANSLATIONS[currentLang] || TRANSLATIONS.en;
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.dataset.i18n;
        if (t[key] !== undefined) el.textContent = t[key];
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.dataset.i18nPlaceholder;
        if (t[key] !== undefined) el.placeholder = t[key];
    });
    const toggle = document.getElementById('lang-toggle');
    if (toggle) toggle.textContent = currentLang === 'en' ? 'BS' : 'EN';
}

function initLangToggle() {
    const btn = document.getElementById('lang-toggle');
    if (!btn) return;
    btn.addEventListener('click', () => {
        currentLang = currentLang === 'en' ? 'bs' : 'en';
        localStorage.setItem('lang', currentLang);
        applyTranslations();
        document.dispatchEvent(new CustomEvent('langchange'));
    });
}

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
        if (user.is_banned) {
            const notice = document.createElement('div');
            notice.className = 'banned-notice';
            notice.textContent = 'Your account has been banned. You can browse content but cannot post or interact.';
            document.querySelector('.container').prepend(notice);
        }

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

    function buildFilterParams() {
        const params = new URLSearchParams();
        const q = searchInput.value.trim();
        if (q) params.set('q', q);
        const city = document.getElementById('filter-city')?.value.trim();
        if (city) params.set('city', city);
        const ownership = document.getElementById('filter-ownership')?.value;
        if (ownership) params.set('ownership', ownership);
        const programs = document.getElementById('filter-programs')?.value.trim();
        if (programs) params.set('programs', programs);
        const minRating = document.getElementById('filter-rating')?.value;
        if (minRating) params.set('min_rating', minRating);
        const minTuition = document.getElementById('filter-tuition-min')?.value;
        if (minTuition) params.set('min_tuition', minTuition);
        const maxTuition = document.getElementById('filter-tuition-max')?.value;
        if (maxTuition) params.set('max_tuition', maxTuition);
        return params.toString();
    }

    async function loadInstitutions() {
        const qs = buildFilterParams();
        const url = '/api/institutions' + (qs ? '?' + qs : '');
        const items = await api('GET', url);
        if (items.length === 0) {
            instList.innerHTML = '<div class="empty-state">No institutions found. Be the first to add one!</div>';
            return;
        }
        instList.innerHTML = items.map(t => {
            const tags = [];
            if (t.city) tags.push(`<span class="meta-chip city-chip">&#128205; ${escapeHtml(t.city)}</span>`);
            if (t.ownership) tags.push(`<span class="meta-chip ownership-chip">${escapeHtml(t.ownership)}</span>`);
            if (t.tuition_min || t.tuition_max) {
                const tRange = t.tuition_max ? `$${t.tuition_min.toLocaleString()}–$${t.tuition_max.toLocaleString()}` : `From $${t.tuition_min.toLocaleString()}`;
                tags.push(`<span class="meta-chip tuition-chip">&#128176; ${tRange}</span>`);
            }
            return `
            <div class="topic-card" onclick="window.location.href='/institution/${t.id}'">
                ${t.cover_image ? `<img src="/${escapeHtml(t.cover_image)}" class="topic-card-cover" alt="">` : ''}
                <h3>${escapeHtml(t.title)}</h3>
                ${t.institution_type ? `<span class="type-badge">${escapeHtml(t.institution_type)}</span>` : ''}
                ${tags.length ? `<div style="margin:0.3rem 0;display:flex;flex-wrap:wrap;gap:0.3rem">${tags.join('')}</div>` : ''}
                <div class="topic-meta">
                    <div>${renderStars(t.avg_rating, t.num_ratings)}</div>
                    <div>by <a href="/profile/${escapeHtml(t.created_by)}" class="username-link" onclick="event.stopPropagation()">${escapeHtml(t.created_by)}</a> &middot; ${timeAgo(t.created_at)}</div>
                </div>
            </div>
        `}).join('');
    }

    loadInstitutions();

    searchBtn.addEventListener('click', () => loadInstitutions());
    searchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') loadInstitutions();
    });
    document.getElementById('filter-btn')?.addEventListener('click', () => loadInstitutions());
    document.getElementById('filter-clear-btn')?.addEventListener('click', () => {
        document.getElementById('filter-city').value = '';
        document.getElementById('filter-ownership').value = '';
        document.getElementById('filter-programs').value = '';
        document.getElementById('filter-rating').value = '';
        document.getElementById('filter-tuition-min').value = '';
        document.getElementById('filter-tuition-max').value = '';
        searchInput.value = '';
        loadInstitutions();
    });

    createBtn.addEventListener('click', async () => {
        createError.textContent = '';
        const title = createInput.value.trim();
        if (!title) return;
        const instType = document.getElementById('create-type').value;
        const description = document.getElementById('create-description').value.trim();
        const emailDomain = document.getElementById('create-email-domain').value.trim();
        const city = document.getElementById('create-city')?.value.trim() || '';
        const ownership = document.getElementById('create-ownership')?.value || '';
        const tuitionMin = parseInt(document.getElementById('create-tuition-min')?.value) || 0;
        const tuitionMax = parseInt(document.getElementById('create-tuition-max')?.value) || 0;
        try {
            const inst = await api('POST', '/api/institutions', { title, institution_type: instType, description, email_domain: emailDomain, city, ownership, tuition_min: tuitionMin, tuition_max: tuitionMax });
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

    // Close login modal
    document.getElementById('close-login-modal')?.addEventListener('click', () => {
        document.getElementById('login-modal').style.display = 'none';
    });

    api('GET', '/api/me').then(user => {
        currentUser = user;
        usernameEl.textContent = user.username;
        usernameEl.style.cursor = 'pointer';
        usernameEl.addEventListener('click', () => {
            window.location.href = '/profile/' + user.username;
        });
        updatePointsBadge(user.contribution_points);
        if (user.is_banned) {
            const notice = document.createElement('div');
            notice.className = 'banned-notice';
            notice.textContent = 'Your account has been banned. You can browse content but cannot post or interact.';
            document.querySelector('.container').prepend(notice);
        }
        loadInstitution();
        loadDiscussion();
        loadLeaderboard().then(() => loadModVerificationPanel());
    }).catch(() => {
        // Guest mode — show the page without login
        if (logoutBtn) { logoutBtn.textContent = 'Login'; logoutBtn.onclick = () => window.location.href = '/'; }
        const guestNotice = document.getElementById('guest-rate-notice');
        if (guestNotice) guestNotice.style.display = 'block';
        const rateFormSection = document.getElementById('rate-form-section');
        if (rateFormSection) rateFormSection.style.display = 'none';
        loadInstitution();
        loadDiscussion();
        loadLeaderboard();
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

            // Extra meta (city, ownership, tuition)
            const extraMeta = document.getElementById('inst-extra-meta');
            if (extraMeta) {
                const hasTuition = t.tuition_min || t.tuition_max;
                const currencyToggleEl = document.getElementById('currency-toggle');
                if (currencyToggleEl) currencyToggleEl.style.display = hasTuition ? 'flex' : 'none';

                const tags = [];
                if (t.city) tags.push(`<span class="meta-chip city-chip">&#128205; ${escapeHtml(t.city)}</span>`);
                if (t.ownership) tags.push(`<span class="meta-chip ownership-chip">${escapeHtml(t.ownership)}</span>`);
                if (hasTuition) {
                    const tRange = formatTuition(t.tuition_min, t.tuition_max, currentCurrency);
                    tags.push(`<span class="meta-chip tuition-chip" id="tuition-chip">&#128176; ${tRange}</span>`);
                }
                // Insert chips before the currency toggle
                const currencyEl = extraMeta.querySelector('#currency-toggle');
                extraMeta.innerHTML = '';
                tags.forEach(tag => { extraMeta.insertAdjacentHTML('beforeend', tag); });
                if (currencyEl) extraMeta.appendChild(currencyEl);
            }

            // Photos gallery
            if (t.photos && t.photos.length > 0) {
                const photosSection = document.getElementById('inst-photos-section');
                const gallery = document.getElementById('inst-photos-gallery');
                photosSection.style.display = 'block';
                const canDelete = currentUser && (currentUser.id === t.created_by_id || currentModeratorId === currentUser.id || currentUser.is_admin);
                gallery.innerHTML = t.photos.map(p => `
                    <div class="gallery-item">
                        <img src="/${escapeHtml(p.path)}" class="gallery-img" onclick="openLightbox('/${escapeHtml(p.path)}')" alt="">
                        ${canDelete ? `<button class="gallery-delete-btn" data-photoid="${p.id}" title="Delete photo">&times;</button>` : ''}
                    </div>
                `).join('');
                gallery.querySelectorAll('.gallery-delete-btn').forEach(btn => {
                    btn.addEventListener('click', async () => {
                        if (!confirm('Delete this photo?')) return;
                        try {
                            await api('DELETE', `/api/institutions/${instId}/photos/${btn.dataset.photoid}`);
                            loadInstitution();
                        } catch (err) { alert(err.message); }
                    });
                });
            } else {
                document.getElementById('inst-photos-section').style.display = 'none';
            }

            // Programs
            if (t.programs) {
                const progsSection = document.getElementById('inst-programs-section');
                const progsList = document.getElementById('inst-programs-list');
                progsSection.style.display = 'block';
                progsList.innerHTML = t.programs.split(',').map(p => p.trim()).filter(Boolean)
                    .map(p => `<span class="program-chip">${escapeHtml(p)}</span>`).join('');
            } else {
                document.getElementById('inst-programs-section').style.display = 'none';
            }

            // Pros / Cons
            if (t.pros || t.cons) {
                document.getElementById('inst-pros-cons-section').style.display = 'block';
                document.getElementById('inst-pros-text').textContent = t.pros || '—';
                document.getElementById('inst-cons-text').textContent = t.cons || '—';
            } else {
                document.getElementById('inst-pros-cons-section').style.display = 'none';
            }

            // Edit panel (creator, mod, admin)
            const canEdit = currentUser && (currentUser.id === t.created_by_id || currentModeratorId === currentUser.id || currentUser.is_admin);
            const editBtnContainer = document.getElementById('inst-edit-btn-container');
            if (canEdit && editBtnContainer) {
                editBtnContainer.style.display = 'block';
                // Pre-fill edit fields
                document.getElementById('edit-name').value = t.title || '';
                document.getElementById('edit-email-domain').value = t.email_domain || '';
                document.getElementById('edit-city').value = t.city || '';
                document.getElementById('edit-ownership').value = t.ownership || '';
                document.getElementById('edit-tuition-min').value = t.tuition_min || '';
                document.getElementById('edit-tuition-max').value = t.tuition_max || '';
                document.getElementById('edit-programs').value = t.programs || '';
                document.getElementById('edit-pros').value = t.pros || '';
                document.getElementById('edit-cons').value = t.cons || '';
            }

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

            // Verification section + rate form visibility
            const verifySection = document.getElementById('verify-section');
            const rateFormSection = document.getElementById('rate-form-section');
            const notVerifiedNotice = document.getElementById('not-verified-notice');
            if (currentUser) {
                if (t.is_current_user_verified) {
                    // Verified: show "you are verified" and show rate form
                    if (verifySection) {
                        verifySection.style.display = 'block';
                        verifySection.innerHTML = '<h3>Verification</h3><p style="color:#22C55E;font-weight:600">&#10003; You are verified for this institution</p>';
                    }
                    if (rateFormSection) rateFormSection.style.display = 'block';
                    if (notVerifiedNotice) notVerifiedNotice.style.display = 'none';
                } else {
                    // Not verified: show verify section, hide rate form
                    if (verifySection) {
                        verifySection.style.display = 'block';
                        if (!t.email_domain) {
                            const emailTab = document.getElementById('verify-tab-email');
                            if (emailTab) emailTab.style.display = 'none';
                            const emailSection = document.getElementById('verify-email-section');
                            if (emailSection) emailSection.style.display = 'none';
                            const photoSection = document.getElementById('verify-photo-section');
                            if (photoSection) photoSection.style.display = 'block';
                        } else {
                            const emailInput = document.getElementById('verify-email');
                            if (emailInput) emailInput.placeholder = `your.name${t.email_domain}`;
                        }
                    }
                    if (rateFormSection) rateFormSection.style.display = 'none';
                    if (notVerifiedNotice) notVerifiedNotice.style.display = 'block';
                }
            }

            if (t.ratings.length === 0) {
                const noRatingsText = (TRANSLATIONS[currentLang] || TRANSLATIONS.en).noRatings;
                ratingsList.innerHTML = `<div class="empty-state">${noRatingsText}</div>`;
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
                    const verifiedText = (TRANSLATIONS[currentLang] || TRANSLATIONS.en).verified;
                    const verifiedBadge = r.is_verified ? `<span class="verified-badge" title="Verified">${verifiedText}</span>` : '';
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

    // Edit panel toggle
    document.getElementById('inst-edit-toggle-btn')?.addEventListener('click', () => {
        const panel = document.getElementById('inst-edit-panel');
        panel.style.display = panel.style.display === 'none' ? 'block' : 'none';
    });

    // Save meta
    document.getElementById('inst-edit-save-btn')?.addEventListener('click', async () => {
        const errEl = document.getElementById('edit-meta-error');
        const sucEl = document.getElementById('edit-meta-success');
        errEl.textContent = ''; sucEl.textContent = '';
        try {
            await api('PATCH', `/api/institutions/${instId}/meta`, {
                title: document.getElementById('edit-name').value.trim(),
                email_domain: document.getElementById('edit-email-domain').value.trim(),
                city: document.getElementById('edit-city').value.trim(),
                ownership: document.getElementById('edit-ownership').value,
                programs: document.getElementById('edit-programs').value.trim(),
                pros: document.getElementById('edit-pros').value.trim(),
                cons: document.getElementById('edit-cons').value.trim(),
                tuition_min: parseInt(document.getElementById('edit-tuition-min').value) || 0,
                tuition_max: parseInt(document.getElementById('edit-tuition-max').value) || 0,
            });
            sucEl.textContent = 'Saved!';
            setTimeout(() => { sucEl.textContent = ''; }, 2000);
            loadInstitution();
        } catch (err) { errEl.textContent = err.message; }
    });

    // Currency toggle
    document.querySelectorAll('.currency-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            currentCurrency = btn.dataset.currency;
            document.querySelectorAll('.currency-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            // Update tuition chip without re-loading full institution
            const tuitionChip = document.getElementById('tuition-chip');
            if (tuitionChip) {
                // Re-load institution to get fresh tuition values rendered in new currency
                loadInstitution();
            }
        });
    });

    // Photo upload
    document.getElementById('photo-upload-input')?.addEventListener('change', async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        const statusEl = document.getElementById('photo-upload-status');
        statusEl.textContent = 'Uploading...';
        const formData = new FormData();
        formData.append('photo', file);
        try {
            const res = await fetch(`/api/institutions/${instId}/photos`, { method: 'POST', body: formData });
            const data = await res.json();
            if (!res.ok) { statusEl.textContent = data.error || 'Upload failed'; }
            else { statusEl.textContent = 'Photo added!'; setTimeout(() => { statusEl.textContent = ''; }, 2000); loadInstitution(); }
        } catch { statusEl.textContent = 'Upload failed'; }
        e.target.value = '';
    });

    // Lightbox
    window.openLightbox = function(src) {
        document.getElementById('lightbox-img').src = src;
        document.getElementById('photo-lightbox').style.display = 'flex';
    };

    // Verify tab toggles
    document.getElementById('verify-tab-email')?.addEventListener('click', () => {
        document.getElementById('verify-tab-email').classList.add('active');
        document.getElementById('verify-tab-photo').classList.remove('active');
        document.getElementById('verify-email-section').style.display = 'block';
        document.getElementById('verify-photo-section').style.display = 'none';
    });
    document.getElementById('verify-tab-photo')?.addEventListener('click', () => {
        document.getElementById('verify-tab-photo').classList.add('active');
        document.getElementById('verify-tab-email').classList.remove('active');
        document.getElementById('verify-photo-section').style.display = 'block';
        document.getElementById('verify-email-section').style.display = 'none';
    });

    // Photo upload handler
    document.getElementById('verify-photo-btn')?.addEventListener('click', async () => {
        const errorEl = document.getElementById('verify-photo-error');
        const successEl = document.getElementById('verify-photo-success');
        errorEl.textContent = '';
        successEl.textContent = '';
        const file = document.getElementById('verify-photo-input').files[0];
        if (!file) { errorEl.textContent = 'Please select a file'; return; }
        const formData = new FormData();
        formData.append('proof', file);
        try {
            const res = await fetch(`/api/institutions/${instId}/verify-photo`, { method: 'POST', body: formData });
            const data = await res.json();
            if (!res.ok) { errorEl.textContent = data.error || 'Upload failed'; } else {
                successEl.textContent = 'Submitted! Your request is awaiting moderator review.';
                document.getElementById('verify-photo-input').value = '';
            }
        } catch (err) {
            errorEl.textContent = 'Upload failed';
        }
    });

    // ==================== Mod Verification Panel ====================
    async function loadModVerificationPanel() {
        const isMod = currentUser && (currentModeratorId === currentUser.id || currentUser.is_admin);
        if (!isMod) return;
        const panel = document.getElementById('mod-verification-panel');
        panel.style.display = 'block';
        const list = document.getElementById('mod-verification-list');
        try {
            const requests = await api('GET', `/api/institutions/${instId}/verification-requests`);
            if (requests.length === 0) {
                list.innerHTML = '<div class="empty-state" style="padding:1rem;font-size:0.9rem">No pending verification requests</div>';
                return;
            }
            list.innerHTML = requests.map(vr => `
                <div class="verification-request-card" data-id="${vr.id}">
                    <div><strong><a href="/profile/${escapeHtml(vr.username)}" class="username-link">${escapeHtml(vr.username)}</a></strong></div>
                    <img src="/${escapeHtml(vr.image_path)}" class="verification-proof-thumb" alt="proof" onclick="window.open('/${escapeHtml(vr.image_path)}', '_blank')">
                    <div style="display:flex;gap:0.5rem;margin-top:0.5rem">
                        <button class="btn btn-accent btn-small vr-approve-btn" data-id="${vr.id}">Approve</button>
                        <button class="btn btn-destructive btn-small vr-reject-btn" data-id="${vr.id}">Reject</button>
                    </div>
                </div>
            `).join('');
            list.querySelectorAll('.vr-approve-btn').forEach(btn => {
                btn.addEventListener('click', async () => {
                    try {
                        await api('PUT', `/api/verification-requests/${btn.dataset.id}`, { status: 'approved' });
                        loadModVerificationPanel();
                        loadInstitution();
                    } catch (err) { alert(err.message); }
                });
            });
            list.querySelectorAll('.vr-reject-btn').forEach(btn => {
                btn.addEventListener('click', async () => {
                    try {
                        await api('PUT', `/api/verification-requests/${btn.dataset.id}`, { status: 'rejected' });
                        loadModVerificationPanel();
                    } catch (err) { alert(err.message); }
                });
            });
        } catch {
            list.innerHTML = '<div class="empty-state" style="padding:1rem;font-size:0.9rem">Failed to load requests</div>';
        }
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

    // Reload dynamic content on language change
    document.addEventListener('langchange', () => {
        if (document.getElementById('inst-title')) loadInstitution();
    });

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
        if (user.is_banned) {
            const notice = document.createElement('div');
            notice.className = 'banned-notice';
            notice.textContent = 'Your account has been banned. You can browse content but cannot post or interact.';
            document.querySelector('.container').prepend(notice);
        }
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

            // Admin link
            initAdminPanel();

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

// ==================== ADMIN PANEL (profile page link) ====================
async function initAdminPanel() {
    const link = document.getElementById('admin-link-section');
    if (link && currentUser && currentUser.is_admin) link.style.display = 'block';
}

// ==================== ADMIN PAGE ====================
function initAdminPage() {
    if (!document.querySelector('.admin-page-tabs')) return;

    const usernameEl = document.getElementById('username');
    const logoutBtn = document.getElementById('logout-btn');

    api('GET', '/api/me').then(user => {
        currentUser = user;
        if (!user.is_admin) { window.location.href = '/home'; return; }
        if (usernameEl) {
            usernameEl.textContent = user.username;
            usernameEl.style.cursor = 'pointer';
            usernameEl.addEventListener('click', () => { window.location.href = '/profile/' + user.username; });
        }
        updatePointsBadge(user.contribution_points);
        loadAdminUsers();
    }).catch(() => window.location.href = '/');

    if (logoutBtn) {
        logoutBtn.addEventListener('click', async () => {
            await api('POST', '/api/logout');
            window.location.href = '/';
        });
    }

    document.querySelectorAll('.admin-page-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('.admin-page-tab').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            document.querySelectorAll('.admin-tab-panel').forEach(p => p.style.display = 'none');
            document.getElementById('admin-tab-' + tab.dataset.tab).style.display = 'block';
            if (tab.dataset.tab === 'verifications') loadAdminVerifications();
            if (tab.dataset.tab === 'bans') loadAdminBans();
        });
    });

    document.getElementById('close-activity-modal')?.addEventListener('click', () => {
        document.getElementById('activity-modal').style.display = 'none';
    });

    async function loadAdminUsers() {
        const list = document.getElementById('admin-users-list');
        try {
            const users = await api('GET', '/api/admin/users');
            if (!users || users.length === 0) {
                list.innerHTML = '<div class="empty-state" style="padding:1rem">No users found</div>';
                return;
            }
            list.innerHTML = users.map(u => `
                <div class="admin-user-row" id="admin-user-row-${u.id}">
                    <div class="admin-user-info">
                        <a href="/profile/${escapeHtml(u.username)}" class="user-name username-link">${escapeHtml(u.username)}</a>
                        <span style="color:#888;font-size:0.85rem">${u.rating_count} ratings</span>
                        ${u.is_banned ? '<span style="color:#EF4444;font-size:0.8rem;font-weight:700">BANNED</span>' : ''}
                    </div>
                    <div style="display:flex;align-items:center;gap:0.5rem;flex-wrap:wrap">
                        <div class="admin-score-control">
                            <span style="font-size:0.85rem;color:#888">pts:</span>
                            <input type="number" value="${u.contribution_points}" data-userid="${u.id}">
                            <button class="btn btn-accent btn-small admin-set-pts-btn" data-userid="${u.id}">Set</button>
                        </div>
                        <button class="btn btn-small" style="background:#6366F1;color:#fff" data-userid="${u.id}" data-username="${escapeHtml(u.username)}" onclick="showActivity(${u.id}, '${escapeHtml(u.username)}')">Activity</button>
                        ${u.is_banned
                            ? `<button class="btn btn-small" style="background:#22C55E;color:#fff" data-userid="${u.id}" onclick="adminUnban(${u.id}, this)">Unban</button>`
                            : `<button class="btn btn-small btn-destructive" data-userid="${u.id}" onclick="adminBan(${u.id}, '${escapeHtml(u.username)}', this)">Ban</button>`
                        }
                    </div>
                </div>
            `).join('');

            list.querySelectorAll('.admin-set-pts-btn').forEach(btn => {
                btn.addEventListener('click', async () => {
                    const input = btn.parentElement.querySelector('input[type="number"]');
                    try {
                        await api('PUT', `/api/admin/users/${btn.dataset.userid}/points`, { points: parseInt(input.value) });
                        btn.textContent = 'Done!';
                        setTimeout(() => { btn.textContent = 'Set'; }, 1000);
                    } catch (err) { alert(err.message); }
                });
            });
        } catch (err) {
            list.innerHTML = '<div class="empty-state" style="padding:1rem">Failed to load users</div>';
        }
    }

    window.showActivity = async function(userId, username) {
        document.getElementById('activity-modal-title').textContent = username + ' — Activity';
        document.getElementById('activity-modal').style.display = 'flex';
        const content = document.getElementById('activity-modal-content');
        content.innerHTML = '<div style="color:#888;text-align:center;padding:1rem">Loading...</div>';
        try {
            const a = await api('GET', `/api/admin/users/${userId}/activity`);
            let html = '';
            html += '<div class="activity-section"><h4>Ratings (' + a.ratings.length + ')</h4>';
            if (a.ratings.length === 0) html += '<div class="activity-item" style="color:#aaa">None</div>';
            else a.ratings.forEach(r => {
                html += `<div class="activity-item"><strong>${escapeHtml(r.title)}</strong> — ${r.score}★ ${r.comment ? '· ' + escapeHtml(r.comment.substring(0, 80)) : ''} <span style="color:#aaa;font-size:0.8rem">${timeAgo(r.created_at)}</span></div>`;
            });
            html += '</div>';
            html += '<div class="activity-section"><h4>Comments (' + a.comments.length + ')</h4>';
            if (a.comments.length === 0) html += '<div class="activity-item" style="color:#aaa">None</div>';
            else a.comments.forEach(c => {
                html += `<div class="activity-item"><strong>${escapeHtml(c.title)}</strong> — ${escapeHtml(c.content.substring(0, 100))} <span style="color:#aaa;font-size:0.8rem">${timeAgo(c.created_at)}</span></div>`;
            });
            html += '</div>';
            html += '<div class="activity-section"><h4>Events (' + a.events.length + ')</h4>';
            if (a.events.length === 0) html += '<div class="activity-item" style="color:#aaa">None</div>';
            else a.events.forEach(ev => {
                html += `<div class="activity-item">${escapeHtml(ev.event_type)} <strong style="color:${ev.points >= 0 ? '#22C55E' : '#EF4444'}">${ev.points > 0 ? '+' : ''}${ev.points} pts</strong> <span style="color:#aaa;font-size:0.8rem">${timeAgo(ev.created_at)}</span></div>`;
            });
            html += '</div>';
            content.innerHTML = html;
        } catch {
            content.innerHTML = '<div style="color:#EF4444;text-align:center;padding:1rem">Failed to load activity</div>';
        }
    };

    window.adminBan = async function(userId, username, btn) {
        const reason = prompt(`Ban reason for ${username}?`, '');
        if (reason === null) return;
        try {
            await api('POST', `/api/admin/users/${userId}/ban`, { reason });
            loadAdminUsers();
        } catch (err) { alert(err.message); }
    };

    window.adminUnban = async function(userId, btn) {
        try {
            const res = await fetch(`/api/admin/users/${userId}/ban`, { method: 'DELETE' });
            const data = await res.json();
            if (!res.ok) throw new Error(data.error);
            loadAdminUsers();
        } catch (err) { alert(err.message); }
    };

    async function loadAdminVerifications() {
        const list = document.getElementById('admin-verifications-list');
        try {
            const reqs = await api('GET', '/api/admin/verification-requests');
            if (reqs.length === 0) {
                list.innerHTML = '<div class="empty-state" style="padding:1rem">No pending verification requests</div>';
                return;
            }
            list.innerHTML = reqs.map(vr => `
                <div class="verification-request-card">
                    <div><strong><a href="/profile/${escapeHtml(vr.username)}" class="username-link">${escapeHtml(vr.username)}</a></strong> — <a href="/institution/${vr.institution_id}" class="username-link">${escapeHtml(vr.institution_name)}</a></div>
                    <img src="/${escapeHtml(vr.image_path)}" class="verification-proof-thumb" alt="proof" onclick="window.open('/${escapeHtml(vr.image_path)}', '_blank')">
                    <div style="display:flex;gap:0.5rem;margin-top:0.5rem">
                        <button class="btn btn-accent btn-small adm-vr-approve" data-id="${vr.id}">Approve</button>
                        <button class="btn btn-destructive btn-small adm-vr-reject" data-id="${vr.id}">Reject</button>
                    </div>
                </div>
            `).join('');
            list.querySelectorAll('.adm-vr-approve').forEach(btn => {
                btn.addEventListener('click', async () => {
                    try {
                        await api('PUT', `/api/verification-requests/${btn.dataset.id}`, { status: 'approved' });
                        loadAdminVerifications();
                    } catch (err) { alert(err.message); }
                });
            });
            list.querySelectorAll('.adm-vr-reject').forEach(btn => {
                btn.addEventListener('click', async () => {
                    try {
                        await api('PUT', `/api/verification-requests/${btn.dataset.id}`, { status: 'rejected' });
                        loadAdminVerifications();
                    } catch (err) { alert(err.message); }
                });
            });
        } catch {
            list.innerHTML = '<div class="empty-state" style="padding:1rem">Failed to load</div>';
        }
    }

    async function loadAdminBans() {
        const list = document.getElementById('admin-bans-list');
        try {
            const bans = await api('GET', '/api/admin/bans');
            if (bans.length === 0) {
                list.innerHTML = '<div class="empty-state" style="padding:1rem">No banned users</div>';
                return;
            }
            list.innerHTML = bans.map(b => `
                <div class="admin-user-row">
                    <div class="admin-user-info">
                        <a href="/profile/${escapeHtml(b.username)}" class="user-name username-link">${escapeHtml(b.username)}</a>
                        <span style="color:#888;font-size:0.85rem">by ${escapeHtml(b.banned_by_name)} &middot; ${escapeHtml(b.reason || 'No reason')} &middot; ${timeAgo(b.created_at)}</span>
                    </div>
                    <button class="btn btn-small" style="background:#22C55E;color:#fff" onclick="adminUnban(${b.user_id}, this)">Unban</button>
                </div>
            `).join('');
        } catch {
            list.innerHTML = '<div class="empty-state" style="padding:1rem">Failed to load</div>';
        }
    }
}

// Init on page load
document.addEventListener('DOMContentLoaded', () => {
    applyTranslations();
    initLangToggle();
    initLoginPage();
    initHomePage();
    initInstitutionPage();
    initProfilePage();
    initAdminPage();
});
