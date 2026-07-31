import { useEffect, useState } from 'react';
import type { AdminUser } from '../api/adminUsers';
import { adminUsersApi } from '../api/adminUsers';
import { useAuth } from '../context/AuthContext';
import UserFormModal from '../components/admin/UserFormModal';
import type { UserFormValues } from '../components/admin/UserFormModal';
import ResetPasswordModal from '../components/admin/ResetPasswordModal';
import EmptyState from '../components/common/EmptyState';

export default function UsersPage() {
  const { user: currentUser } = useAuth();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState('');

  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<AdminUser | null>(null);
  const [resettingPassword, setResettingPassword] = useState<AdminUser | null>(null);

  const load = () => {
    setLoading(true);
    adminUsersApi
      .list(1, 100)
      .then(({ users }) => setUsers(users))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const handleCreate = async (values: UserFormValues) => {
    await adminUsersApi.create({
      name: values.name,
      email: values.email,
      password: values.password,
      role: values.role,
    });
    load();
  };

  const handleUpdate = async (values: UserFormValues) => {
    if (!editing) return;
    await adminUsersApi.update(editing.id, {
      name: values.name,
      email: values.email,
      role: values.role,
    });
    load();
  };

  const handleResetPassword = async (password: string) => {
    if (!resettingPassword) return;
    await adminUsersApi.resetPassword(resettingPassword.id, password);
  };

  const handleDelete = async (u: AdminUser) => {
    if (!confirm(`Delete user "${u.name}"? This action cannot be undone.`)) return;
    setErrorMsg('');
    try {
      await adminUsersApi.delete(u.id);
      load();
    } catch (err: unknown) {
      const message =
        (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error
          ?.message || 'Failed to delete user';
      setErrorMsg(message);
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6 gap-3">
        <h1 className="text-2xl font-bold text-gray-900">Users</h1>
        <button
          onClick={() => setCreateOpen(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
        >
          Add User
        </button>
      </div>

      {errorMsg && (
        <div className="bg-red-50 text-red-600 p-3 rounded mb-4 text-sm">{errorMsg}</div>
      )}

      {loading ? (
        <div className="text-gray-500">Loading users...</div>
      ) : users.length === 0 ? (
        <EmptyState title="No users" description="Add the first user to get started" />
      ) : (
        <>
          {/* Desktop table */}
          <div className="hidden md:block bg-white shadow rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-gray-600 text-left">
                <tr>
                  <th className="px-4 py-3 font-medium">Name</th>
                  <th className="px-4 py-3 font-medium">Email</th>
                  <th className="px-4 py-3 font-medium">Role</th>
                  <th className="px-4 py-3 font-medium">Joined</th>
                  <th className="px-4 py-3 font-medium text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {users.map((u) => {
                  const isSelf = u.id === currentUser?.id;
                  return (
                    <tr key={u.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-gray-900">
                        {u.name}
                        {isSelf && (
                          <span className="ml-2 text-xs text-gray-400">(you)</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-gray-600">{u.email}</td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-block px-2 py-0.5 text-xs rounded ${
                            u.role === 'admin'
                              ? 'bg-purple-100 text-purple-700'
                              : 'bg-gray-100 text-gray-700'
                          }`}
                        >
                          {u.role}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-500">
                        {new Date(u.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="inline-flex gap-2">
                          <button
                            onClick={() => setEditing(u)}
                            className="text-blue-600 hover:text-blue-800"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() => setResettingPassword(u)}
                            className="text-gray-600 hover:text-gray-900"
                          >
                            Reset PW
                          </button>
                          <button
                            onClick={() => handleDelete(u)}
                            disabled={isSelf}
                            className="text-red-600 hover:text-red-800 disabled:opacity-40 disabled:cursor-not-allowed"
                            title={isSelf ? 'You cannot delete your own account' : ''}
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          {/* Mobile cards */}
          <div className="md:hidden flex flex-col gap-3">
            {users.map((u) => {
              const isSelf = u.id === currentUser?.id;
              return (
                <div key={u.id} className="bg-white shadow rounded-lg p-4">
                  <div className="flex justify-between items-start mb-2">
                    <div>
                      <div className="font-semibold text-gray-900">
                        {u.name}
                        {isSelf && (
                          <span className="ml-2 text-xs text-gray-400">(you)</span>
                        )}
                      </div>
                      <div className="text-sm text-gray-500">{u.email}</div>
                    </div>
                    <span
                      className={`px-2 py-0.5 text-xs rounded ${
                        u.role === 'admin'
                          ? 'bg-purple-100 text-purple-700'
                          : 'bg-gray-100 text-gray-700'
                      }`}
                    >
                      {u.role}
                    </span>
                  </div>
                  <div className="text-xs text-gray-400 mb-3">
                    Joined {new Date(u.created_at).toLocaleDateString()}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button
                      onClick={() => setEditing(u)}
                      className="px-3 py-1.5 text-sm border border-gray-300 rounded hover:bg-gray-50"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => setResettingPassword(u)}
                      className="px-3 py-1.5 text-sm border border-gray-300 rounded hover:bg-gray-50"
                    >
                      Reset Password
                    </button>
                    <button
                      onClick={() => handleDelete(u)}
                      disabled={isSelf}
                      className="px-3 py-1.5 text-sm text-red-600 border border-red-200 rounded hover:bg-red-50 disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </>
      )}

      <UserFormModal
        isOpen={createOpen}
        mode="create"
        onClose={() => setCreateOpen(false)}
        onSubmit={handleCreate}
      />

      <UserFormModal
        isOpen={!!editing}
        mode="edit"
        initial={editing}
        disableRoleChange={editing?.id === currentUser?.id}
        onClose={() => setEditing(null)}
        onSubmit={handleUpdate}
      />

      <ResetPasswordModal
        isOpen={!!resettingPassword}
        user={resettingPassword}
        onClose={() => setResettingPassword(null)}
        onSubmit={handleResetPassword}
      />
    </div>
  );
}
