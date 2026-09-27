// Mirrors frontend/src/context/LanguageContext.jsx: nested-key t() with {{param}} interpolation,
// Hebrew by default, choice persisted on the device.
import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';
import en from './translations/en';
import he from './translations/he';

export const translations = { en, he };
export const LANGUAGE_KEY = 'language';
export const DEFAULT_LANGUAGE = 'he';

export const translate = (language, key, params = {}) => {
    let value = translations[language];
    for (const k of key.split('.')) {
        if (value == null) return key;
        value = value[k];
    }
    if (typeof value !== 'string') return key;
    return Object.entries(params).reduce(
        (str, [param, val]) => str.replace(new RegExp(`{{${param}}}`, 'g'), String(val)),
        value
    );
};

const LanguageContext = createContext(null);

export const LanguageProvider = ({ children, initialLanguage }) => {
    const [language, setLanguageState] = useState(initialLanguage || DEFAULT_LANGUAGE);

    useEffect(() => {
        if (initialLanguage) return;
        AsyncStorage.getItem(LANGUAGE_KEY).then((saved) => {
            if (saved && translations[saved]) setLanguageState(saved);
        });
    }, [initialLanguage]);

    const setLanguage = useCallback((lang) => {
        setLanguageState(lang);
        AsyncStorage.setItem(LANGUAGE_KEY, lang);
    }, []);

    const toggleLanguage = useCallback(() => {
        setLanguage(language === 'he' ? 'en' : 'he');
    }, [language, setLanguage]);

    const t = useCallback((key, params) => translate(language, key, params), [language]);

    const value = useMemo(() => ({
        language,
        isRTL: language === 'he',
        locale: language === 'he' ? 'he-IL' : 'en-US',
        t,
        setLanguage,
        toggleLanguage,
    }), [language, t, setLanguage, toggleLanguage]);

    return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>;
};

export const useLanguage = () => {
    const ctx = useContext(LanguageContext);
    if (!ctx) throw new Error('useLanguage must be used inside LanguageProvider');
    return ctx;
};

// RTL without I18nManager.forceRTL (which needs an app restart): screens apply these
// on top of their LTR styles so switching language flips the layout instantly.
export const useRTL = () => {
    const { isRTL } = useLanguage();
    return useMemo(() => ({
        isRTL,
        row: { flexDirection: isRTL ? 'row-reverse' : 'row' },
        text: { textAlign: isRTL ? 'right' : 'left', writingDirection: isRTL ? 'rtl' : 'ltr' },
        alignStart: { alignItems: isRTL ? 'flex-end' : 'flex-start' },
        arrowBack: isRTL ? '→' : '←',
    }), [isRTL]);
};
