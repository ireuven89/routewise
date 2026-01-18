import { useState, useEffect } from 'react';
import { jobsAPI, customersAPI, techniciansAPI } from '../api/client';
import Layout from '../components/Layout';
import { format } from 'date-fns';

const Jobs = () => {
    const [jobs, setJobs] = useState([]);
    const [customers, setCustomers] = useState([]);
    const [technicians, setTechnicians] = useState([]);
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState('all'); // all, scheduled, in_progress, completed
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
                techniciansAPI.getAll(true), // active only
            ]);

            setJobs(jobsRes.data || []);
            setCustomers(customersRes.data || []);
            setTechnicians(techniciansRes.data || []);
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
            alert('Failed to create job');
        }
    };

    const handleUpdateJob = async (jobData) => {
        try {
            await jobsAPI.update(editingJob.id, jobData);
            await loadData();
            setEditingJob(null);
        } catch (error) {
            console.error('Failed to update job:', error);
            alert('Failed to update job');
        }
    };

    const handleAssignTechnician = async (jobId, technicianId) => {
        try {
            await jobsAPI.assignTechnician(jobId, technicianId);
            await loadData();
        } catch (error) {
            console.error('Failed to assign technician:', error);
            alert('Failed to assign technician');
        }
    };

    const handleUpdateStatus = async (jobId, status) => {
        try {
            await jobsAPI.updateStatus(jobId, status);
            await loadData();
        } catch (error) {
            console.error('Failed to update status:', error);
            alert('Failed to update status');
        }
    };

    const handleDeleteJob = async (jobId) => {
        if (!window.confirm('Are you sure you want to delete this job?')) return;

        try {
            await jobsAPI.delete(jobId);
            await loadData();
        } catch (error) {
            console.error('Failed to delete job:', error);
            alert('Failed to delete job');
        }
    };

    const filteredJobs = jobs.filter(job => {
        if (filter === 'all') return true;
        return job.status === filter;
    });

    if (loading) {
        return (
            <Layout>
                <div className="flex justify-center items-center h-64">
                    <div className="text-lg text-gray-600">Loading jobs...</div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                {/* Header */}
                <div className="flex justify-between items-center mb-6">
                    <h1 className="text-3xl font-bold text-gray-900">Jobs</h1>
                    <button
                        onClick={() => setShowCreateModal(true)}
                        className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium"
                    >
                        + Create Job
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
                            {status === 'all' ? 'All' : status.replace('_', ' ').toUpperCase()}
                        </button>
                    ))}
                </div>

                {/* Jobs List */}
                {filteredJobs.length === 0 ? (
                    <div className="bg-white shadow rounded-lg p-8 text-center">
                        <p className="text-gray-500">No jobs found. Create your first job!</p>
                    </div>
                ) : (
                    <div className="bg-white shadow overflow-hidden rounded-lg">
                        <ul className="divide-y divide-gray-200">
                            {filteredJobs.map(job => (
                                <JobItem
                                    key={job.id}
                                    job={job}
                                    technicians={technicians}
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
                        technicians={technicians}
                        onSave={handleCreateJob}
                        onClose={() => setShowCreateModal(false)}
                    />
                )}

                {/* Edit Job Modal */}
                {editingJob && (
                    <JobModal
                        job={editingJob}
                        customers={customers}
                        technicians={technicians}
                        onSave={handleUpdateJob}
                        onClose={() => setEditingJob(null)}
                    />
                )}
            </div>
        </Layout>
    );
};

// Job Item Component
const JobItem = ({ job, technicians, onEdit, onDelete, onAssignTechnician, onUpdateStatus }) => {
    const statusColors = {
        scheduled: 'bg-blue-100 text-blue-800',
        in_progress: 'bg-yellow-100 text-yellow-800',
        completed: 'bg-green-100 text-green-800',
        cancelled: 'bg-red-100 text-red-800',
    };

    const getTechnicianName = (techId) => {
        const tech = technicians.find(t => t.id === techId);
        return tech ? tech.name : 'Unassigned';
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
                                Scheduled: {format(new Date(job.scheduled_at), 'PPp')}
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

                    <div className="mt-3 flex items-center space-x-4">
                        {/* Status Badge */}
                        <span className={`px-3 py-1 rounded-full text-xs font-medium ${statusColors[job.status]}`}>
              {job.status.replace('_', ' ').toUpperCase()}
            </span>

                        {/* Technician Assignment */}
                        <select
                            value={job.technician_id || ''}
                            onChange={(e) => onAssignTechnician(job.id, e.target.value ? parseInt(e.target.value) : null)}
                            className="text-sm border-gray-300 rounded-md"
                        >
                            <option value="">Unassigned</option>
                            {technicians.map(tech => (
                                <option key={tech.id} value={tech.id}>
                                    {tech.name}
                                </option>
                            ))}
                        </select>

                        {/* Status Update */}
                        {job.status === 'scheduled' && (
                            <button
                                onClick={() => onUpdateStatus(job.id, 'in_progress')}
                                className="text-sm bg-yellow-600 hover:bg-yellow-700 text-white px-3 py-1 rounded-md"
                            >
                                Start Job
                            </button>
                        )}
                        {job.status === 'in_progress' && (
                            <button
                                onClick={() => onUpdateStatus(job.id, 'completed')}
                                className="text-sm bg-green-600 hover:bg-green-700 text-white px-3 py-1 rounded-md"
                            >
                                Complete
                            </button>
                        )}

                        {/* Edit/Delete */}
                        <button
                            onClick={onEdit}
                            className="text-sm text-blue-600 hover:text-blue-800"
                        >
                            Edit
                        </button>
                        <button
                            onClick={onDelete}
                            className="text-sm text-red-600 hover:text-red-800"
                        >
                            Delete
                        </button>
                    </div>
                </div>
            </div>
        </li>
    );
};

// Job Modal Component
const JobModal = ({ job, customers, technicians, onSave, onClose }) => {
    const [formData, setFormData] = useState({
        customer_id: job?.customer_id || '',
        technician_id: job?.technician_id || '',
        title: job?.title || '',
        description: job?.description || '',
        scheduled_at: job?.scheduled_at ? job.scheduled_at.slice(0, 16) : '',
        duration_minutes: job?.duration_minutes || 60,
        price: job?.price || '',
    });

    const handleChange = (e) => {
        const value = e.target.type === 'number' ? parseFloat(e.target.value) : e.target.value;
        setFormData({
            ...formData,
            [e.target.name]: value,
        });
    };

    const handleSubmit = (e) => {
        e.preventDefault();

        const jobData = {
            ...formData,
            customer_id: parseInt(formData.customer_id),
            technician_id: formData.technician_id ? parseInt(formData.technician_id) : null,
            price: formData.price ? parseFloat(formData.price) : null,
            scheduled_at: new Date(formData.scheduled_at).toISOString(),
        };

        onSave(jobData);
    };

    return (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto">
                <div className="px-6 py-4 border-b border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900">
                        {job ? 'Edit Job' : 'Create New Job'}
                    </h2>
                </div>

                <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
                    {/* Customer */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            Customer *
                        </label>
                        <select
                            name="customer_id"
                            value={formData.customer_id}
                            onChange={handleChange}
                            required
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        >
                            <option value="">Select a customer</option>
                            {customers.map(customer => (
                                <option key={customer.id} value={customer.id}>
                                    {customer.name} - {customer.address}
                                </option>
                            ))}
                        </select>
                    </div>

                    {/* Technician */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            Technician (optional)
                        </label>
                        <select
                            name="technician_id"
                            value={formData.technician_id}
                            onChange={handleChange}
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        >
                            <option value="">Unassigned</option>
                            {technicians.map(tech => (
                                <option key={tech.id} value={tech.id}>
                                    {tech.name}
                                </option>
                            ))}
                        </select>
                    </div>

                    {/* Title */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            Job Title *
                        </label>
                        <input
                            type="text"
                            name="title"
                            value={formData.title}
                            onChange={handleChange}
                            required
                            placeholder="AC Repair"
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>

                    {/* Description */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            Description
                        </label>
                        <textarea
                            name="description"
                            value={formData.description}
                            onChange={handleChange}
                            rows={3}
                            placeholder="Unit not cooling properly..."
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>

                    {/* Scheduled Date/Time */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            Scheduled Date & Time *
                        </label>
                        <input
                            type="datetime-local"
                            name="scheduled_at"
                            value={formData.scheduled_at}
                            onChange={handleChange}
                            required
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>

                    {/* Duration */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            Duration (minutes)
                        </label>
                        <input
                            type="number"
                            name="duration_minutes"
                            value={formData.duration_minutes}
                            onChange={handleChange}
                            min="15"
                            step="15"
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>

                    {/* Price */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            Price ($)
                        </label>
                        <input
                            type="number"
                            name="price"
                            value={formData.price}
                            onChange={handleChange}
                            min="0"
                            step="0.01"
                            placeholder="150.00"
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>

                    {/* Buttons */}
                    <div className="flex justify-end space-x-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                        >
                            {job ? 'Update Job' : 'Create Job'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default Jobs;