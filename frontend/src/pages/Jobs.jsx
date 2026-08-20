import { useState, useEffect } from 'react';
import { jobsAPI, customersAPI, workersAPI, paymentAPI } from '../api/client';
import Layout from '../components/Layout';
import { format } from 'date-fns';
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';
import JobModal from '../components/JobModal';
import { FaBriefcase, FaCalendarAlt, FaWrench } from 'react-icons/fa';

const Jobs = () => {
    const { t } = useLanguage();
    const { organization } = useAuth();
    const industry = organization?.industry || 'hvac';

    const [jobs, setJobs] = useState([]);
    const [customers, setCustomers] = useState([]);
    const [workers, setWorkers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState('all');
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [editingJob, setEditingJob] = useState(null);

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        try {
            const [jobsRes, customersRes, techniciansRes] = await Promise.all([
                jobsAPI.getAll(),
                customersAPI.getAll(),
                workersAPI.getAll(true),
            ]);
            setJobs(jobsRes.data || []);
            setCustomers(customersRes.data || []);
            setWorkers(techniciansRes.data || []);
            setLoading(false);
        } catch (error) {
            console.error('Failed to load data:', error);
            setLoading(false);
        }
    };

    const handleCreateJob = async (jobData) => {
        try {
            await jobsAPI.create(jobData);
            await loadData();
            setShowCreateModal(false);
        } catch (error) {
            console.error('Failed to create job:', error);
            alert(t('jobs.failedCreate'));
        }
    };

    const handleUpdateJob = async (jobData) => {
        try {
            await jobsAPI.update(editingJob.id, jobData);
            await loadData();
            setEditingJob(null);
        } catch (error) {
            console.error('Failed to update job:', error);
            alert(t('jobs.failedUpdate'));
        }
    };

    const handleAssignTechnician = async (jobId, technicianId) => {
        try {
            await jobsAPI.assignTechnician(jobId, technicianId);
            await loadData();
        } catch (error) {
            console.error('Failed to assign technician:', error);
            alert(t('jobs.failedAssign'));
        }
    };

    const handleUpdateStatus = async (jobId, status) => {
        try {
            await jobsAPI.updateStatus(jobId, status);
            await loadData();
        } catch (error) {
            console.error('Failed to update status:', error);
            alert(t('jobs.failedStatus'));
        }
    };

    const handleDeleteJob = async (jobId) => {
        if (!window.confirm(t('jobs.deleteConfirm'))) return;
        try {
            await jobsAPI.delete(jobId);
            await loadData();
        } catch (error) {
            console.error('Failed to delete job:', error);
            alert(t('jobs.failedDelete'));
        }
    };

    const filteredJobs = jobs.filter(job => filter === 'all' || job.status === filter);

    const filterLabels = {
        all: t('jobs.filterAll'),
        scheduled: t('status.scheduled'),
        in_progress: t('status.inProgress'),
        completed: t('status.completed'),
        cancelled: t('status.cancelled'),
    };

    if (loading) {
        return (
            <Layout>
                <div className="px-4 sm:px-0">
                    <div className="flex justify-between items-center mb-6">
                        <div className="h-8 w-32 bg-gray-200 rounded-lg animate-pulse" />
                        <div className="h-10 w-32 bg-gray-200 rounded-xl animate-pulse" />
                    </div>
                    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
                        {[1, 2, 3].map(i => (
                            <div key={i} className="flex items-start gap-4 px-6 py-4 border-b border-gray-50 last:border-0">
                                <div className="w-10 h-10 rounded-full bg-gray-200 animate-pulse flex-shrink-0" />
                                <div className="flex-1 space-y-2">
                                    <div className="h-4 w-48 bg-gray-200 rounded animate-pulse" />
                                    <div className="h-3 w-32 bg-gray-100 rounded animate-pulse" />
                                    <div className="h-3 w-40 bg-gray-100 rounded animate-pulse" />
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
                        <h1 className="text-2xl font-bold text-gray-900">{t(`industry.${industry}.jobs`)}</h1>
                        <span className="text-sm font-medium text-gray-400 bg-gray-100 px-2.5 py-0.5 rounded-full">
                            {filteredJobs.length}
                        </span>
                    </div>
                    <button
                        onClick={() => setShowCreateModal(true)}
                        className="bg-[#ff6b35] hover:opacity-90 text-white px-4 py-2.5 rounded-xl text-sm font-semibold shadow-sm transition-opacity"
                    >
                        {t('jobs.createJob')}
                    </button>
                </div>

                {/* Filters */}
                <div className="mb-6 flex flex-wrap gap-1 bg-white rounded-2xl border border-gray-100 shadow-sm p-2 w-fit">
                    {['all', 'scheduled', 'in_progress', 'completed', 'cancelled'].map(status => (
                        <button
                            key={status}
                            onClick={() => setFilter(status)}
                            className={`px-4 py-1.5 rounded-full text-sm font-medium transition-colors ${
                                filter === status
                                    ? 'bg-[#1e3a5f] text-white'
                                    : 'text-gray-600 hover:bg-gray-50'
                            }`}
                        >
                            {filterLabels[status]}
                        </button>
                    ))}
                </div>

                {/* Jobs List */}
                {filteredJobs.length === 0 ? (
                    <div className="bg-white rounded-2xl shadow-sm border border-gray-100">
                        <div className="px-6 py-12 text-center">
                            <div className="w-14 h-14 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                                <FaBriefcase className="w-6 h-6 text-gray-300" />
                            </div>
                            <p className="text-sm font-medium text-gray-500">{t('jobs.noJobs')}</p>
                            <button
                                onClick={() => setShowCreateModal(true)}
                                className="mt-4 inline-flex items-center gap-1.5 text-sm font-semibold text-blue-600 hover:text-blue-700"
                            >
                                + {t('jobs.createJob')}
                            </button>
                        </div>
                    </div>
                ) : (
                    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
                        <ul className="divide-y divide-gray-50">
                            {filteredJobs.map(job => (
                                <JobItem
                                    key={job.id}
                                    job={job}
                                    technicians={workers}
                                    onEdit={() => setEditingJob(job)}
                                    onDelete={() => handleDeleteJob(job.id)}
                                    onAssignTechnician={handleAssignTechnician}
                                    onUpdateStatus={handleUpdateStatus}
                                />
                            ))}
                        </ul>
                    </div>
                )}

                {showCreateModal && (
                    <JobModal
                        customers={customers}
                        technicians={workers}
                        onSave={handleCreateJob}
                        onClose={() => setShowCreateModal(false)}
                    />
                )}

                {editingJob && (
                    <JobModal
                        job={editingJob}
                        customers={customers}
                        technicians={workers}
                        onSave={handleUpdateJob}
                        onClose={() => setEditingJob(null)}
                    />
                )}
            </div>
        </Layout>
    );
};

// ─── JobItem ──────────────────────────────────────────────────────────────────
const JobItem = ({ job, technicians, onEdit, onDelete, onAssignTechnician, onUpdateStatus }) => {
    const { t } = useLanguage();

    const statusConfig = {
        scheduled: { bg: 'bg-blue-50', text: 'text-blue-700' },
        in_progress: { bg: 'bg-amber-50', text: 'text-amber-700' },
        completed: { bg: 'bg-emerald-50', text: 'text-emerald-700' },
        cancelled: { bg: 'bg-red-50', text: 'text-red-700' },
    };

    const statusLabels = {
        scheduled: t('status.scheduled'),
        in_progress: t('status.inProgress'),
        completed: t('status.completed'),
        cancelled: t('status.cancelled'),
    };

    const { bg, text } = statusConfig[job.status] || statusConfig.scheduled;

    return (
        <li className="flex items-start gap-4 px-6 py-4 hover:bg-gray-50 transition-colors duration-100">
            <div className="w-10 h-10 rounded-full bg-blue-50 flex items-center justify-center flex-shrink-0 mt-0.5">
                <FaWrench className="w-4 h-4 text-blue-600" />
            </div>

            <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0">
                        <h3 className="text-sm font-semibold text-gray-900 truncate">{job.title}</h3>
                        <p className="text-xs text-gray-500 mt-0.5">
                            {job.customer?.name}
                            {job.customer?.phone && (
                                <span className="text-gray-400"> · {job.customer.phone}</span>
                            )}
                        </p>
                        <p className="text-xs text-gray-400 mt-0.5 flex items-center gap-1">
                            <FaCalendarAlt className="w-3 h-3 flex-shrink-0" />
                            {format(new Date(job.scheduled_at), 'PPp')}
                        </p>
                        {job.description && (
                            <p className="text-xs text-gray-400 mt-1 truncate">{job.description}</p>
                        )}
                    </div>
                    {job.price && (
                        <span className="text-sm font-bold text-gray-900 flex-shrink-0">
                            ₪{job.price.toFixed(0)}
                        </span>
                    )}
                </div>

                <div className="mt-3 flex flex-wrap items-center gap-2">
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold ${bg} ${text}`}>
                        {statusLabels[job.status]}
                    </span>

                    <select
                        value={job.technician_id || ''}
                        onChange={(e) => onAssignTechnician(job.id, e.target.value ? parseInt(e.target.value) : null)}
                        className="text-xs border border-gray-200 rounded-lg px-2 py-1 bg-white text-gray-600 focus:outline-none focus:ring-1 focus:ring-gray-300"
                    >
                        <option value="">{t('jobs.unassigned')}</option>
                        {technicians.map(tech => (
                            <option key={tech.id} value={tech.id}>{tech.name}</option>
                        ))}
                    </select>

                    {job.status === 'scheduled' && (
                        <button
                            onClick={() => onUpdateStatus(job.id, 'in_progress')}
                            className="text-xs bg-amber-500 hover:bg-amber-600 text-white px-3 py-1 rounded-lg font-semibold transition-colors"
                        >
                            {t('jobs.startJob')}
                        </button>
                    )}
                    {job.status === 'in_progress' && (
                        <button
                            onClick={() => onUpdateStatus(job.id, 'completed')}
                            className="text-xs bg-emerald-500 hover:bg-emerald-600 text-white px-3 py-1 rounded-lg font-semibold transition-colors"
                        >
                            {t('jobs.complete')}
                        </button>
                    )}

                    <PaymentBadge job={job} />

                    <button
                        onClick={onEdit}
                        className="text-xs font-medium text-[#1e3a5f] hover:opacity-70 transition-opacity ml-1"
                    >
                        {t('jobs.edit')}
                    </button>
                    <button
                        onClick={onDelete}
                        className="text-xs font-medium text-red-500 hover:text-red-700 transition-colors"
                    >
                        {t('jobs.delete')}
                    </button>
                </div>
            </div>
        </li>
    );
};

// ─── PaymentBadge ─────────────────────────────────────────────────────────────
const PaymentBadge = ({ job }) => {
    const { t } = useLanguage();
    const [notification, setNotification] = useState(null);
    const [loaded, setLoaded] = useState(false);
    const [busy, setBusy] = useState(false);

    const eligible = job.status === 'completed' && job.price;

    useEffect(() => {
        if (!eligible) return;
        let cancelled = false;
        paymentAPI.getNotifications(job.id)
            .then(res => {
                if (cancelled) return;
                const list = res.data || [];
                setNotification(list[0] || null);
                setLoaded(true);
            })
            .catch(() => setLoaded(true));
        return () => { cancelled = true; };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [job.id, job.status, job.price]);

    if (!eligible || !loaded) return null;

    const handleSend = async () => {
        setBusy(true);
        try {
            const res = await paymentAPI.sendRequest(job.id);
            setNotification(res.data);
        } catch (error) {
            alert(error.response?.data?.error || t('jobs.paymentSendFailed'));
        } finally {
            setBusy(false);
        }
    };

    const handleMarkPaid = async () => {
        setBusy(true);
        try {
            await paymentAPI.markPaid(notification.id);
            setNotification({ ...notification, payment_status: 'paid' });
        } catch (error) {
            alert(error.response?.data?.error || t('jobs.markPaidFailed'));
        } finally {
            setBusy(false);
        }
    };

    if (!notification || notification.payment_status === 'send_failed') {
        return (
            <button
                onClick={handleSend}
                disabled={busy}
                className="text-xs bg-orange-500 hover:bg-orange-600 text-white px-3 py-1 rounded-lg font-semibold transition-colors disabled:opacity-50"
            >
                {t('jobs.sendPaymentRequest')}
            </button>
        );
    }

    if (notification.payment_status === 'paid') {
        return (
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700">
                {t('jobs.paymentPaid')}
            </span>
        );
    }

    return (
        <>
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-50 text-amber-700">
                {t('jobs.paymentSent')}
            </span>
            <button
                onClick={handleMarkPaid}
                disabled={busy}
                className="text-xs bg-emerald-500 hover:bg-emerald-600 text-white px-3 py-1 rounded-lg font-semibold transition-colors disabled:opacity-50"
            >
                {t('jobs.markPaid')}
            </button>
        </>
    );
};

export default Jobs;
