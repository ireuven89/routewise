import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import en from '../translations/en';
import he from '../translations/he';

const translations = { en, he };

const LanguageContext = createContext();

export const LanguageProvider = ({ children }) => {
    const [language, setLanguage] = useState(() => {
        return localStorage.getItem('language') || 'he'; // default Hebrew for Israeli market
    });

    const isRTL = language === 'he';

    // Set dir attribute on <html> whenever language changes
    useEffect(() => {
        document.documentElement.setAttribute('dir', isRTL ? 'rtl' : 'ltr');
        document.documentElement.setAttribute('lang', language);
        localStorage.setItem('language', language);
    }, [language, isRTL]);

    // t('dashboard.title') → looks up nested key
    // t('dashboard.companyCodeDesc', { workers: 'עובדים' }) → interpolates {{workers}}
    const t = useCallback((key, params = {}) => {
        const keys = key.split('.');
        let value = translations[language];

        for (const k of keys) {
            if (value == null) return key; // fallback: return the key itself
            value = value[k];
        }

        if (typeof value !== 'string') return key;

        // Replace {{param}} placeholders
        return Object.entries(params).reduce(
            (str, [param, val]) => str.replace(new RegExp(`{{${param}}}`, 'g'), val),
            value
        );
    }, [language]);

    const toggleLanguage = useCallback(() => {
        setLanguage((prev) => (prev === 'he' ? 'en' : 'he'));
    }, []);

    return (
        <LanguageContext.Provider value={{ language, isRTL, t, toggleLanguage }}>
            {children}
        </LanguageContext.Provider>
    );
};

export const useLanguage = () => {
    const context = useContext(LanguageContext);
    if (!context) throw new Error('useLanguage must be used inside LanguageProvider');
    return context;
};
