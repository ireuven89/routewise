import { useState, useEffect } from 'react';
import { workersAPI } from '../api/client';
import Layout from '../components/Layout';
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';
import WorkerModal from '../components/WorkerModal';

const Workers = () => {
    const { t } = useLanguage();
    const { organization } = useAuth();
    const industry = organization?.industry || 'hvac';

    const [workers, setWorkers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showModal, setShowModal] = useState(false);
    const [editingWorker, setEditingWorker] = useState(null);
    const [showActiveOnly, setShowActiveOnly] = useState(true);

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

    if (loading) {
        return (
            <Layout>
                <div className="flex justify-center items-center h-64">
                    <div className="text-lg text-gray-600">{t('workers.loading')}</div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                {/* Header */}
                <div className="flex justify-between items-center mb-6">
                    <h1 className="text-3xl font-bold text-gray-900">{t(`industry.${industry}.workers`)}</h1>
                    <button
                        onClick={() => setShowModal(true)}
                        style={{ backgroundColor: '#ff6b35' }}
                        className="hover:opacity-90 text-white px-4 py-2 rounded-md font-medium"
                    >
                        {t(`industry.${industry}.addWorker`)}
                    </button>
                </div>

                {/* Filter */}
                <div className="mb-6">
                    <label className="flex items-center">
                        <input
                            type="checkbox"
                            checked={showActiveOnly}
                            onChange={(e) => setShowActiveOnly(e.target.checked)}
                            className="rounded border-gray-300"
                            style={{ accentColor: '#1e3a5f' }}
                        />
                        <span className="ml-2 text-sm text-gray-700">{t('workers.showActiveOnly')}</span>
                    </label>
                </div>

                {/* Workers List */}
                {workers.length === 0 ? (
                    <div className="bg-white shadow rounded-lg p-8 text-center">
                        <p className="text-gray-500">{t('workers.noWorkers')}</p>
                    </div>
                ) : (
                    <div className="bg-white shadow overflow-hidden rounded-lg">
                        <ul className="divide-y divide-gray-200">
                            {workers.map((worker) => (
                                <li key={worker.id} className="px-6 py-4 hover:bg-gray-50">
                                    <div className="flex items-center justify-between">
                                        <div className="flex-1">
                                            <div className="flex items-center">
                                                <h3 className="text-lg font-medium text-gray-900">{worker.name}</h3>
                                                {worker.is_active ? (
                                                    <span className="ml-3 px-2 py-1 text-xs font-medium bg-green-100 text-green-800 rounded-full">
                                                        {t('workers.active')}
                                                    </span>
                                                ) : (
                                                    <span className="ml-3 px-2 py-1 text-xs font-medium bg-gray-100 text-gray-800 rounded-full">
                                                        {t('workers.inactive')}
                                                    </span>
                                                )}
                                                {worker.role && (
                                                    <span
                                                        className="ml-3 px-2 py-1 text-xs font-medium rounded-full text-white"
                                                        style={{ backgroundColor: '#1e3a5f' }}
                                                    >
                                                        {worker.role}
                                                    </span>
                                                )}
                                            </div>
                                            <p className="text-sm text-gray-500 mt-1">
                                                📞 {worker.phone}
                                                {worker.email && ` • ✉️ ${worker.email}`}
                                            </p>
                                            {worker.home_address && (
                                                <p className="text-sm text-gray-600 mt-1">
                                                    🏠 {worker.home_address}
                                                </p>
                                            )}
                                        </div>
                                        <div className="flex gap-3">
                                            <button
                                                onClick={() => setEditingWorker(worker)}
                                                style={{ color: '#1e3a5f' }}
                                                className="hover:opacity-70 font-medium"
                                            >
                                                {t('jobs.edit')}
                                            </button>
                                            <button
                                                onClick={() => handleDelete(worker.id)}
                                                className="text-red-600 hover:text-red-800 font-medium"
                                            >
                                                {t('jobs.delete')}
                                            </button>
                                        </div>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    </div>
                )}

                {/* Create Modal */}
                {showModal && <WorkerModal onSave={handleCreate} onClose={() => setShowModal(false)} />}

                {/* Edit Modal */}
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
