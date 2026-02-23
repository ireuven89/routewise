import { useState, useEffect } from 'react';
import { jobsAPI, customersAPI, workersAPI } from '../api/client';
import Layout from '../components/Layout';
import { format } from 'date-fns';
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';
import JobModal from '../components/JobModal';

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

    const filteredJobs = jobs.filter(job => {
        if (filter === 'all') return true;
        return job.status === filter;
    });

    // Filter button labels
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
                <div className="flex justify-center items-center h-64">
                    <div className="text-lg text-gray-600">{t('jobs.loading')}</div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                {/* Header */}
                <div className="flex justify-between items-center mb-6">
                    <h1 className="text-3xl font-bold text-gray-900">{t(`industry.${industry}.jobs`)}</h1>
                    <button
                        onClick={() => setShowCreateModal(true)}
                        className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium"
                    >
                        {t('jobs.createJob')}
                    </button>
                </div>

                {/* Filters */}
                <div className="flex space-x-2 mb-6">
                    {['all', 'scheduled', 'in_progress', 'completed', 'cancelled'].map(status => (
                        <button
                            key={status}
                            onClick={() => setFilter(status)}
                            className={`px-4 py-2 rounded-md font-medium ${
                                filter === status
                                    ? 'bg-blue-600 text-white'
                                    : 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
                            }`}
                        >
                            {filterLabels[status]}
                        </button>
                    ))}
                </div>

                {/* Jobs List */}
                {filteredJobs.length === 0 ? (
                    <div className="bg-white shadow rounded-lg p-8 text-center">
                        <p className="text-gray-500">{t('jobs.noJobs')}</p>
                    </div>
                ) : (
                    <div className="bg-white shadow overflow-hidden rounded-lg">
                        <ul className="divide-y divide-gray-200">
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

                {/* Create Job Modal */}
                {showCreateModal && (
                    <JobModal
                        customers={customers}
                        technicians={workers}
                        onSave={handleCreateJob}
                        onClose={() => setShowCreateModal(false)}
                    />
                )}

                {/* Edit Job Modal */}
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

    const statusColors = {
        scheduled: 'bg-blue-100 text-blue-800',
        in_progress: 'bg-yellow-100 text-yellow-800',
        completed: 'bg-green-100 text-green-800',
        cancelled: 'bg-red-100 text-red-800',
    };

    const statusLabels = {
        scheduled: t('status.scheduled'),
        in_progress: t('status.inProgress'),
        completed: t('status.completed'),
        cancelled: t('status.cancelled'),
    };

    return (
        <li className="px-6 py-4 hover:bg-gray-50">
            <div className="flex items-center justify-between">
                <div className="flex-1">
                    <div className="flex items-center justify-between">
                        <div>
                            <h3 className="text-lg font-medium text-gray-900">{job.title}</h3>
                            <p className="text-sm text-gray-500 mt-1">
                                {job.customer?.name} • {job.customer?.address}
                            </p>
                            <p className="text-sm text-gray-500">
                                {t('jobs.scheduled')} {format(new Date(job.scheduled_at), 'PPp')}
                            </p>
                            {job.description && (
                                <p className="text-sm text-gray-600 mt-2">{job.description}</p>
                            )}
                        </div>
                        <div className="flex items-center space-x-4">
                            {job.price && (
                                <span className="text-lg font-semibold text-gray-900">
                                    ${job.price.toFixed(2)}
                                </span>
                            )}
                        </div>
                    </div>

                    <div className="mt-3 flex items-center gap-4">
                        {/* Status Badge */}
                        <span className={`px-3 py-1 rounded-full text-xs font-medium ${statusColors[job.status]}`}>
                            {statusLabels[job.status]}
                        </span>

                        {/* Technician Assignment */}
                        <select
                            value={job.technician_id || ''}
                            onChange={(e) => onAssignTechnician(job.id, e.target.value ? parseInt(e.target.value) : null)}
                            className="text-sm border-gray-300 rounded-md"
                        >
                            <option value="">{t('jobs.unassigned')}</option>
                            {technicians.map(tech => (
                                <option key={tech.id} value={tech.id}>
                                    {tech.name}
                                </option>
                            ))}
                        </select>

                        {/* Status Update Buttons */}
                        {job.status === 'scheduled' && (
                            <button
                                onClick={() => onUpdateStatus(job.id, 'in_progress')}
                                className="text-sm bg-yellow-600 hover:bg-yellow-700 text-white px-3 py-1 rounded-md"
                            >
                                {t('jobs.startJob')}
                            </button>
                        )}
                        {job.status === 'in_progress' && (
                            <button
                                onClick={() => onUpdateStatus(job.id, 'completed')}
                                className="text-sm bg-green-600 hover:bg-green-700 text-white px-3 py-1 rounded-md"
                            >
                                {t('jobs.complete')}
                            </button>
                        )}

                        {/* Edit/Delete */}
                        <button onClick={onEdit} className="text-sm text-blue-600 hover:text-blue-800">
                            {t('jobs.edit')}
                        </button>
                        <button onClick={onDelete} className="text-sm text-red-600 hover:text-red-800">
                            {t('jobs.delete')}
                        </button>
                    </div>
                </div>
            </div>
        </li>
    );
};

export default Jobs;
