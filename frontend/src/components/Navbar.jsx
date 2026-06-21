import { Link, useNavigate } from 'react-router-dom';
import { FaGlobe } from 'react-icons/fa';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';

const Navbar = () => {
    const { user, organization, logout } = useAuth();
    const { t, language, toggleLanguage } = useLanguage();
    const navigate = useNavigate();

    const industry = organization?.industry || 'hvac';

    const handleLogout = () => {
        logout();
        navigate('/login');
    };

    return (
        <nav className="bg-white shadow-sm">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                <div className="flex justify-between h-16">
                    <div className="flex items-center gap-6">
                        <div className="flex-shrink-0 flex items-center">
                            <Link to="/dashboard" className="text-xl font-bold" style={{ color: '#1e3a5f' }}>
                                RouteWise
                            </Link>
                        </div>
                        <div className="hidden sm:flex sm:gap-8">
                            <Link
                                to="/dashboard"
                                className="border-transparent text-gray-900 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
                            >
                                {t('nav.dashboard')}
                            </Link>
                            <Link
                                to="/jobs"
                                className="border-transparent text-gray-900 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
                            >
                                {t(`industry.${industry}.jobs`)}
                            </Link>
                            <Link
                                to="/customers"
                                className="border-transparent text-gray-900 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
                            >
                                {t('nav.customers')}
                            </Link>
                            <Link
                                to="/technicians"
                                className="border-transparent text-gray-900 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
                            >
                                {t(`industry.${industry}.workers`)}
                            </Link>
                            <Link
                                to="/settings"
                                className="border-transparent text-gray-900 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
                            >
                                {t('nav.settings')}
                            </Link>
                        </div>
                    </div>
                    <div className="flex items-center gap-4">
                        {/* Language toggle */}
                        <button
                            onClick={toggleLanguage}
                            className="flex items-center gap-1.5 text-xs font-medium text-gray-500 hover:text-gray-700 transition-colors"
                        >
                            <FaGlobe className="w-3.5 h-3.5" />
                            {language === 'he' ? 'English' : 'עברית'}
                        </button>

                        <span className="text-sm text-gray-700">{organization?.name || user?.company_name}</span>

                        <button
                            onClick={handleLogout}
                            className="bg-white text-gray-700 hover:bg-gray-50 px-4 py-2 rounded-md text-sm font-medium border border-gray-300"
                        >
                            {t('nav.logout')}
                        </button>
                    </div>
                </div>
            </div>
        </nav>
    );
};

export default Navbar;
