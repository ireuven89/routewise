import { useState, useEffect } from 'react';
import { workersAPI } from '../api/client';
import Layout from '../components/Layout';
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';
import WorkerModal from '../components/WorkerModal';
import { FaHardHat, FaPhone, FaEnvelope, FaHome, FaSearch } from 'react-icons/fa';

const getInitials = (name) => {
    if (!name) return '?';
    const parts = name.trim().split(/\s+/);
    if (parts.length === 1) return parts[0][0].toUpperCase();
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
};

const Workers = () => {
    const { t } = useLanguage();
    const { organization } = useAuth();
    const industry = organization?.industry || 'hvac';

    const [workers, setWorkers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showModal, setShowModal] = useState(false);
    const [editingWorker, setEditingWorker] = useState(null);
    const [showActiveOnly, setShowActiveOnly] = useState(true);
    const [search, setSearch] = useState('');

    useEffect(() => {
        loadWorkers();
    }, [showActiveOnly]);

    const loadWorkers = async () => {
        try {
            const response = await workersAPI.getAll(showActiveOnly);
            setWorkers(response.data || []);
            setLoading(false);
        } catch (error) {
            console.error('Failed to load workers:', error);
            setLoading(false);
        }
    };

    const handleCreate = async (workerData) => {
        try {
            await workersAPI.create(workerData);
            await loadWorkers();
            setShowModal(false);
        } catch (error) {
            console.error('Failed to create worker:', error);
            alert(t('workers.failedCreate'));
        }
    };

    const handleUpdate = async (workerData) => {
        try {
            await workersAPI.update(editingWorker.id, workerData);
            await loadWorkers();
            setEditingWorker(null);
        } catch (error) {
            console.error('Failed to update worker:', error);
            alert(t('workers.failedUpdate'));
        }
    };

    const handleDelete = async (workerId) => {
        if (!window.confirm(t('workers.deleteConfirm'))) return;
        try {
            await workersAPI.delete(workerId);
            await loadWorkers();
        } catch (error) {
            console.error('Failed to delete worker:', error);
            alert(t('workers.failedDelete'));
        }
    };

    const filteredWorkers = workers.filter(w =>
        w.name?.toLowerCase().includes(search.toLowerCase())
    );

    if (loading) {
        return (
            <Layout>
                <div className="px-4 sm:px-0">
                    <div className="flex justify-between items-center mb-6">
                        <div className="h-8 w-40 bg-gray-200 rounded-lg animate-pulse" />
                        <div className="h-10 w-36 bg-gray-200 rounded-xl animate-pulse" />
                    </div>
                    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
                        {[1, 2, 3].map(i => (
                            <div key={i} className="flex items-center gap-4 px-6 py-4 border-b border-gray-50 last:border-0">
                                <div className="w-10 h-10 rounded-full bg-gray-200 animate-pulse flex-shrink-0" />
                                <div className="flex-1 space-y-2">
                                    <div className="h-4 w-32 bg-gray-200 rounded animate-pulse" />
                                    <div className="h-3 w-48 bg-gray-100 rounded animate-pulse" />
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                {/* Header */}
                <div className="flex justify-between items-center mb-6">
                    <div className="flex items-center gap-3">
                        <h1 className="text-2xl font-bold text-gray-900">{t(`industry.${industry}.workers`)}</h1>
                        <span className="text-sm font-medium text-gray-400 bg-gray-100 px-2.5 py-0.5 rounded-full">
                            {filteredWorkers.length}
                        </span>
                    </div>
                    <button
                        onClick={() => setShowModal(true)}
                        className="bg-[#ff6b35] hover:opacity-90 text-white px-4 py-2.5 rounded-xl text-sm font-semibold shadow-sm transition-opacity"
                    >
                        {t(`industry.${industry}.addWorker`)}
                    </button>
                </div>

                {/* Filter & Search Bar */}
                <div className="mb-6 bg-white rounded-2xl border border-gray-100 shadow-sm px-4 py-3 flex flex-wrap items-center gap-4">
                    <div className="relative flex-1 min-w-[180px] max-w-xs">
                        <FaSearch className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400 pointer-events-none" />
                        <input
                            type="text"
                            value={search}
                            onChange={e => setSearch(e.target.value)}
                            placeholder="Search..."
                            className="w-full pl-9 pr-3 py-1.5 text-sm border border-gray-200 rounded-lg bg-gray-50 focus:outline-none focus:ring-2 focus:ring-[#1e3a5f] focus:ring-opacity-20 focus:bg-white"
                        />
                    </div>
                    <label className="flex items-center gap-2 cursor-pointer flex-shrink-0">
                        <input
                            type="checkbox"
                            checked={showActiveOnly}
                            onChange={(e) => setShowActiveOnly(e.target.checked)}
                            className="w-4 h-4 rounded border-gray-300 accent-[#1e3a5f]"
                        />
                        <span className="text-sm text-gray-600">{t('workers.showActiveOnly')}</span>
                    </label>
                </div>

                {/* Workers List */}
                {filteredWorkers.length === 0 ? (
                    <div className="bg-white rounded-2xl shadow-sm border border-gray-100">
                        <div className="px-6 py-12 text-center">
                            <div className="w-14 h-14 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                                <FaHardHat className="w-6 h-6 text-gray-300" />
                            </div>
                            <p className="text-sm font-medium text-gray-500">{t('workers.noWorkers')}</p>
                            <button
                                onClick={() => setShowModal(true)}
                                className="mt-4 inline-flex items-center gap-1.5 text-sm font-semibold text-blue-600 hover:text-blue-700"
                            >
                                + {t(`industry.${industry}.addWorker`)}
                            </button>
                        </div>
                    </div>
                ) : (
                    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
                        <ul className="divide-y divide-gray-50">
                            {filteredWorkers.map(worker => (
                                <li key={worker.id} className="flex items-center gap-4 px-6 py-4 hover:bg-gray-50 transition-colors duration-100">
                                    {/* Avatar */}
                                    <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center flex-shrink-0">
                                        <span className="text-xs font-bold text-white">{getInitials(worker.name)}</span>
                                    </div>

                                    {/* Info */}
                                    <div className="flex-1 min-w-0">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <span className="text-sm font-semibold text-gray-900 truncate">{worker.name}</span>

                                            <span className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold ${
                                                worker.is_active
                                                    ? 'bg-emerald-50 text-emerald-700'
                                                    : 'bg-gray-100 text-gray-500'
                                            }`}>
                                                <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${
                                                    worker.is_active ? 'bg-emerald-400' : 'bg-gray-300'
                                                }`} />
                                                {worker.is_active ? t('workers.active') : t('workers.inactive')}
                                            </span>

                                            {worker.role && (
                                                <span className="bg-[#1e3a5f] text-white px-2.5 py-0.5 rounded-full text-xs font-semibold">
                                                    {worker.role}
                                                </span>
                                            )}
                                        </div>

                                        <div className="mt-1 flex flex-wrap gap-x-4 gap-y-0.5">
                                            <span className="text-xs text-gray-500 flex items-center gap-1">
                                                <FaPhone className="w-3 h-3 text-gray-400 flex-shrink-0" />
                                                {worker.phone}
                                            </span>
                                            {worker.email && (
                                                <span className="text-xs text-gray-500 flex items-center gap-1">
                                                    <FaEnvelope className="w-3 h-3 text-gray-400 flex-shrink-0" />
                                                    {worker.email}
                                                </span>
                                            )}
                                            {worker.home_address && (
                                                <span className="text-xs text-gray-400 flex items-center gap-1">
                                                    <FaHome className="w-3 h-3 flex-shrink-0" />
                                                    {worker.home_address}
                                                </span>
                                            )}
                                        </div>
                                    </div>

                                    {/* Actions */}
                                    <div className="flex items-center gap-3 flex-shrink-0">
                                        <button
                                            onClick={() => setEditingWorker(worker)}
                                            className="text-xs font-medium text-[#1e3a5f] hover:opacity-70 transition-opacity"
                                        >
                                            {t('jobs.edit')}
                                        </button>
                                        <button
                                            onClick={() => handleDelete(worker.id)}
                                            className="text-xs font-medium text-red-500 hover:text-red-700 transition-colors"
                                        >
                                            {t('jobs.delete')}
                                        </button>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    </div>
                )}

                {showModal && <WorkerModal onSave={handleCreate} onClose={() => setShowModal(false)} />}
                {editingWorker && (
                    <WorkerModal
                        worker={editingWorker}
                        onSave={handleUpdate}
                        onClose={() => setEditingWorker(null)}
                    />
                )}
            </div>
        </Layout>
    );
};

export default Workers;
