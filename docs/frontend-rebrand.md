# RouteWise Frontend Rebrand - Quick Guide

**Time:** 4-6 hours  
**Colors:** Navy (#1e3a5f) + Orange (#ff6b35)

---

## 🎨 Theme Setup

**Create `src/theme.js`:**
```javascript
export const colors = {
  primary: '#1e3a5f',
  accent: '#ff6b35',
  lightBg: '#f5f7fa',
  inputBg: '#e8eef5',
  white: '#ffffff',
  text: '#1a1a1a',
  textGray: '#6b7280',
  success: '#10b981',
  info: '#3b82f6',
};
```

**Update `src/index.css`:**
```css
:root {
  --primary: #1e3a5f;
  --accent: #ff6b35;
  --light-bg: #f5f7fa;
}

body {
  background: var(--light-bg);
}
```

---

## 🔧 Component Updates

### Buttons
```css
.btn-primary {
  background: #1e3a5f;
  color: white;
}

.btn-accent {
  background: #ff6b35;
  color: white;
}
```

### Header/Navbar
```css
.navbar {
  background: #1e3a5f;
  color: white;
}
```

### Cards
```css
.card {
  background: white;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}
```

### Status Badges
```css
.badge-in-progress { background: #ff6b35; color: white; }
.badge-completed { background: #10b981; color: white; }
.badge-scheduled { background: #3b82f6; color: white; }
```

---

## ✅ Quick Checklist

- [ ] Add theme.js
- [ ] Update CSS variables
- [ ] Change navbar to navy
- [ ] Change buttons (primary=navy, CTA=orange)
- [ ] Update status badges
- [ ] Change page backgrounds to #f5f7fa
- [ ] Test all pages

---

**Done. Keep it simple.**
